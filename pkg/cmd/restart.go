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
	"reflect"

	"github.com/Ridecell/ridectl/pkg/exec"
	"github.com/Ridecell/ridectl/pkg/kubernetes"
	utils "github.com/Ridecell/ridectl/pkg/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(rollingRestartCmd)
}

var rollingRestartCmd = &cobra.Command{
	Use:   "restart [flags] <cluster_name> <pod_type>",
	Short: "Performs a rolling restart of pods.",
	Long:  `Restarts all pods of a certain type (web|celeryd|etc).`,
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("Cluster name argument is required.")
		}
		if len(args) == 1 {
			return fmt.Errorf("Deployment type argument is required.")
		}
		if len(args) > 2 {
			return fmt.Errorf("Too many arguments")
		}
		return nil
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		binaryExists := utils.CheckBinary("kubectl")
		if !binaryExists {
			return fmt.Errorf("kubectl is not installed. Follow the instructions here: https://kubernetes.io/docs/tasks/tools/#kubectl to install it")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		kubeconfig := utils.GetKubeconfig()
		target, err := kubernetes.ParseSubject(args[0])
		if err != nil {
			return errors.Wrapf(err, "not a valid target %s", args[0])
		}

		kubeObj := kubernetes.GetAppropriateObjectWithContext(*kubeconfig, args[0], target)
		if reflect.DeepEqual(kubeObj, kubernetes.Kubeobject{}) {
			return errors.Wrapf(err, "no instance found %s", args[0])
		}

		kubectlArgs := []string{"kubectl", "rollout", "restart", "deployment", "-n", target.Namespace, fmt.Sprintf("%s-%s", target.Name, args[1]), "--context", kubeObj.Context.Cluster}
		return exec.Exec(kubectlArgs)
	},
}
