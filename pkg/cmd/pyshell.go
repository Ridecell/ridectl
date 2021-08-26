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
	"flag"
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/Ridecell/ridectl/pkg/exec"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/util/homedir"

	kubernetes "github.com/Ridecell/ridectl/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
)

func init() {
	rootCmd.AddCommand(pyShellCmd)
}

var pyShellCmd = &cobra.Command{
	Use:   "pyshell [flags] <cluster_name>",
	Short: "Open a Python shell on a Summon instance",
	Long:  `Open an interactive Python terminal on a Summon instance running on Kubernetes`,
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("Cluster name argument is required")
		}
		if len(args) > 1 {
			return fmt.Errorf("Too many arguments")
		}
		return nil
	},
	RunE: func(_ *cobra.Command, args []string) error {
		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()
		target, err := kubernetes.ParseSubject(args[0])
		if err != nil {
			return errors.Wrap(err, "not a valid target")
		}

		podLabels := make(map[string]string)
		if target.Type == "summon" {
			podLabels["app.kubernetes.io/instance"] = fmt.Sprintf("%s-web", args[0])
		} else if target.Type == "microservice" {
			fmt.Printf("this is target: %+v", target)
			podLabels["app"] = fmt.Sprintf("%s-svc-%s", target.Env, target.Namespace)
			podLabels["environment"] = target.Env
			podLabels["region"] = target.Region
			podLabels["role"] = "web"
		} else {
			return fmt.Errorf("Cannot find pod without knowing the target's type: %#v", target)
		}

		kubeObj := kubernetes.GetAppropriateObjectWithContext(*kubeconfig, args[0], "pyshell", podLabels)
		if reflect.DeepEqual(kubeObj, kubernetes.Kubeobject{}) {
			return fmt.Errorf("No instance found")
		}
		pod := kubeObj.Object.(*corev1.Pod)
		// Spawn kubectl exec.
		kubectlArgs := []string{"kubectl", "exec", "-it", "-n", pod.Namespace, pod.Name, "--context", kubeObj.Context.Cluster, "--", "bash", "-l", "-c", "python manage.py shell"}
		return exec.Exec(kubectlArgs)

	},
}
