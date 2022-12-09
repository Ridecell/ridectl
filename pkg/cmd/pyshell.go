/*
Copyright 2021 Ridecell, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/Ridecell/ridectl/pkg/exec"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubernetes "github.com/Ridecell/ridectl/pkg/kubernetes"
	utils "github.com/Ridecell/ridectl/pkg/utils"
	corev1 "k8s.io/api/core/v1"
)

func init() {
	rootCmd.AddCommand(pyShellCmd)
}

var pyShellCmd = &cobra.Command{
	Use:   "pyshell [flags] <cluster_name>",
	Short: "Open a Python shell on a Summon instance",
	Long: "Open an interactive Python shell for a Summon instance or microservice running on Kubernetes.\n" +
		"For summon instances: pyshell <tenant>-<env>                   -- e.g. ridectl pyshell darwin-qa\n" +
		"For microservices: pyshell svc-<region>-<env>-<microservice>   -- e.g. ridectl pyshell svc-us-master-dispatch",
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("cluster name argument is required")
		}
		if len(args) > 1 {
			return fmt.Errorf("too many arguments")
		}
		return nil
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		utils.CheckTshLogin()
		utils.CheckKubectl()
		return nil
	},
	RunE: func(_ *cobra.Command, args []string) error {
		kubeconfig := utils.GetKubeconfig()
		target, err := kubernetes.ParseSubject(args[0])
		if err != nil {
			pterm.Error.Println(err, "Its not a valid Summonplatform or Microservice")
			os.Exit(1)
		}

		podLabels := make(map[string]string)
		if target.Type == "summon" {
			podLabels["app.kubernetes.io/instance"] = fmt.Sprintf("%s-web", args[0])
		} else if target.Type == "microservice" {
			podLabels["app"] = fmt.Sprintf("%s-svc-%s", target.Env, target.Namespace)
			podLabels["environment"] = target.Env
			podLabels["region"] = target.Region
			podLabels["role"] = "web"
		}

		kubeObj := kubernetes.GetAppropriateObjectWithContext(*kubeconfig, args[0], target, inCluster)
		if reflect.DeepEqual(kubeObj, kubernetes.Kubeobject{}) {
			pterm.Error.Printf("No instance found %s\n", args[0])
			os.Exit(1)
		}

		labelSet := labels.Set{}
		for k, v := range podLabels {
			labelSet[k] = v
		}

		listOptions := &client.ListOptions{
			Namespace:     target.Namespace,
			LabelSelector: labels.SelectorFromSet(labelSet),
		}

		podList := &corev1.PodList{}
		err = kubeObj.Client.List(context.Background(), podList, listOptions)
		if err != nil {
			pterm.Error.Printf("instance not found in %s", kubeObj.Context)
			os.Exit(1)
		}
		if len(podList.Items) < 1 {
			pterm.Error.Printf("instance not found in %s", kubeObj.Context)
			os.Exit(1)
		}

		pod := corev1.Pod{}
		for _, po := range podList.Items {
			// choose only first running pod
			if kubernetes.IsContainerReady(&po.Status) {
				pod = po
				break
			}
		}
		if pod.Name == "" {
			pterm.Error.Printf("no running pod found in %s", kubeObj.Context)
			os.Exit(1)
		}

		// Spawn kubectl exec.
		pterm.Info.Printf("Connecting to %s/%s\n", pod.Namespace, pod.Name)

		// Warn people that this is a container.
		pterm.Warning.Printf("Remember that this is a container and most changes will have no effect\n")

		kubectlArgs := []string{"exec", "-it", "-n", pod.Namespace, pod.Name, "--context", kubeObj.Context, "--", "bash", "-l", "-c", "python manage.py shell"}
		return exec.ExecuteCommand("kubectl", kubectlArgs, true)

	},
}
