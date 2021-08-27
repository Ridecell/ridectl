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
	"fmt"
	"os/exec"
	"reflect"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	ridectlexec "github.com/Ridecell/ridectl/pkg/exec"
	kubernetes "github.com/Ridecell/ridectl/pkg/kubernetes"
	utils "github.com/Ridecell/ridectl/pkg/utils"
	corev1 "k8s.io/api/core/v1"
)

func init() {
	rootCmd.AddCommand(shellCmd)
}

var shellCmd = &cobra.Command{
	Use:   "shell [flags] <cluster_name>",
	Short: "Open a shell on a Summon instance or microservice",
	Long:  `Open an interactive bash terminal on a Summon instance or microservice running on Kubernetes`,
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
		_, err := exec.LookPath("kubectl")
		if err != nil {
			return errors.Wrap(err, "Unable to find kubectl")
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
		} else {
			return fmt.Errorf("cannot find pod without knowing the target's type: %#v", target)
		}

		kubeObj := kubernetes.GetAppropriateObjectWithContext(*kubeconfig, args[0], target, podLabels)
		if reflect.DeepEqual(kubeObj, kubernetes.Kubeobject{}) {
			return fmt.Errorf("no instance found")
		}

		pod := kubeObj.Object.(*corev1.Pod)
		// Spawn kubectl exec.

		fmt.Printf("Connecting to %s/%s\n", pod.Namespace, pod.Name)
		// Warn people that this is a container.
		fmt.Printf("Remember that this is a container and most changes will have no effect\n")

		kubectlArgs := []string{"kubectl", "exec", "-it", "-n", pod.Namespace, pod.Name, "--context", kubeObj.Context.Cluster, "--", "bash", "-l"}
		return ridectlexec.Exec(kubectlArgs)

	},
}
