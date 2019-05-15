/*
Copyright 2019 Ridecell, Inc.

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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/Ridecell/ridectl/pkg/kubernetes"

	corev1 "k8s.io/api/core/v1"
)

func init() {
	rootCmd.AddCommand(passwordCmd)
}

var passwordCmd = &cobra.Command{
	Use:   "password [flags] <cluster_name>",
	Short: "Gets dispatcher password from a Summon Instance",
	Long:  `Returns dispatcher django password from a Summon Instance Secret`,
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
		secretName := fmt.Sprintf("%s-dispatcher.django-password", args[0])
		namespace := kubernetes.ParseNamespace(args[0])

		fetchObject := &kubernetes.KubeObject{Top: &corev1.Secret{}}
		err := kubernetes.GetObject(kubeconfigFlag, secretName, namespace, fetchObject)
		if err != nil {
			return errors.Wrap(err, "unable to find secret")
		}
		secret, ok := fetchObject.Top.(*corev1.Secret)
		if !ok {
			return errors.New("unable to convert to secret object")
		}

		fmt.Printf("Password for %s: %s\n", args[0], string(secret.Data["password"]))
		return nil
	},
}
