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
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubernetes "github.com/Ridecell/ridectl/pkg/kubernetes"
	utils "github.com/Ridecell/ridectl/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func init() {
	rootCmd.AddCommand(shellCmd)
}

var shellCmd = &cobra.Command{
	Use:   "shell [flags] <cluster_name>",
	Short: "Open a shell on a Summon instance or microservice",
	Long: "Open an interactive Bash shell for a Summon instance or microservice running on Kubernetes.\n" +
		"For summon instances: shell <tenant>-<env>                   -- e.g. ridectl shell darwin-qa\n" +
		"For microservices: shell svc-<region>-<env>-<microservice>   -- e.g. ridectl shell svc-us-master-dispatch",
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			pterm.Error.Println("cluster name argument is required")
			os.Exit(1)
		}
		if len(args) > 1 {
			pterm.Error.Println("too many arguments")
			os.Exit(1)
		}
		return nil
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		utils.CheckVPN()
		binaryExists := utils.CheckBinary("kubectl")
		if !binaryExists {
			pterm.Error.Printf("kubectl is not installed. Follow the instructions here: https://kubernetes.io/docs/tasks/tools/#kubectl to install it\n")
			os.Exit(1)
		}
		return nil
	},
	RunE: func(_ *cobra.Command, args []string) error {
		kubeconfig := utils.GetKubeconfig()
		target, err := kubernetes.ParseSubject(args[0])
		if err != nil {
			pterm.Error.Println(err, "Its not a valid target")
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

		kubeObj := kubernetes.GetAppropriateObjectWithContext(*kubeconfig, args[0], target)
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
			return fmt.Errorf("instance not found in %s", kubeObj.Context.Cluster)
		}
		if len(podList.Items) < 1 {
			return fmt.Errorf("instance not found in %s", kubeObj.Context.Cluster)
		}

		pod := podList.Items[0]
		// Spawn kubectl exec.
		pterm.Info.Printf("Connecting to %s/%s\n", pod.Namespace, pod.Name)

		// Warn people that this is a container.
		pterm.Warning.Printf("Remember that this is a container and most changes will have no effect\n")

		kubectlArgs := []string{"kubectl", "exec", "-it", "-n", pod.Namespace, pod.Name, "--context", kubeObj.Context.Cluster, "--", "bash", "-l"}
		return exec.Exec(kubectlArgs)

	},
}
