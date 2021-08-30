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
	"reflect"

	"github.com/Ridecell/ridectl/pkg/exec"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubernetes "github.com/Ridecell/ridectl/pkg/kubernetes"
	utils "github.com/Ridecell/ridectl/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
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
		binaryExists := utils.CheckBinary("kubectl1")
		if !binaryExists {
			return fmt.Errorf("kubectl is not installed. Follow the instructions here: https://kubernetes.io/docs/tasks/tools/#kubectl to install it")
		}
		return nil
	},
	RunE: func(_ *cobra.Command, args []string) error {
		kubeconfig := utils.GetKubeconfig()
		target, err := kubernetes.ParseSubject(args[0])
		if err != nil {
			return errors.Wrap(err, "not a valid target")
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
			return fmt.Errorf("no instance found")
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
		kubectlArgs := []string{"kubectl", "exec", "-it", "-n", pod.Namespace, pod.Name, "--context", kubeObj.Context.Cluster, "--", "bash", "-l", "-c", "python manage.py shell"}
		return exec.Exec(kubectlArgs)

	},
}
