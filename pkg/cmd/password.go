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
	"strings"

	"github.com/Ridecell/ridectl/pkg/kubernetes"
	"github.com/Ridecell/ridectl/pkg/utils"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
)

var readOnlyUserFlag bool

func init() {
	rootCmd.AddCommand(passwordCmd)
	passwordCmd.Flags().BoolVar(&readOnlyUserFlag, "readonly", false, "get connection details for readonly user")
}

var passwordCmd = &cobra.Command{
	Use:   "password [flags] <cluster_name>",
	Short: "Gets dispatcher password from a Summon Instance",
	Long:  `Returns dispatcher django password from a Summon Instance Secret`,
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
		utils.CheckVPN()
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		kubeconfig := utils.GetKubeconfig()
		target, err := kubernetes.ParseSubject(args[0])
		if err != nil {
			return errors.Wrapf(err, "not a valid target %s", args[0])
		}

		kubeObj := kubernetes.GetAppropriateObjectWithContext(*kubeconfig, args[0], target)
		if reflect.DeepEqual(kubeObj, kubernetes.Kubeobject{}) {
			return errors.Wrapf(err, "no instance found %s", args[0])
		}

		secret := &corev1.Secret{}
		if !readOnlyUserFlag {
			err = kubeObj.Client.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s.django-password", args[0]), Namespace: target.Namespace}, secret)
			if err != nil {
				return errors.Wrapf(err, "error getting secret  for instance %s", args[0])
			}

			fmt.Printf("Password for %s: %s\n", args[0], string(secret.Data["password"]))
		} else {
			// get a list of secrets which have readonly in their name
			readOnlysecrets := []string{}
			secrets := &corev1.SecretList{}
			err = kubeObj.Client.List(ctx, secrets)
			if err != nil {
				return errors.Wrapf(err, "error getting secrets for instance %s", args[0])
			}
			for _, secret := range secrets.Items {
				//add the readonly secrets to the list if name contains readonly
				if strings.Contains(secret.Name, "-readonly.postgres-user-password") {
					readOnlysecrets = append(readOnlysecrets, secret.Name)
				}
			}
			if len(readOnlysecrets) == 0 {
				return errors.Wrapf(err, "no readonly secrets found for instance %s", args[0])
			}
			// prompt user to select a readonly secret
			prompt := promptui.Select{
				Label: "Select secret",
				Items: readOnlysecrets,
			}
			_, result, err := prompt.Run()
			if err != nil {
				return errors.Wrapf(err, "Prompt failed")
			}
			// get the password from the selected secret
			err = kubeObj.Client.Get(ctx, types.NamespacedName{Name: result, Namespace: target.Namespace}, secret)
			if err != nil {
				return errors.Wrapf(err, "error getting secret  for instance %s", args[0])
			}
			fmt.Printf("Readonly User Connection Details\n================\n")
			fmt.Printf("Database Type: Postgres\n") // Hard code-y
			fmt.Printf("Database Host: %s\n", string(secret.Data["host"]))
			fmt.Printf("Database Port: %s\n", string(secret.Data["port"]))
			fmt.Printf("Database Name: %s\n", string(secret.Data["dbname"]))
			fmt.Printf("Database Username: %s\n", string(secret.Data["username"]))
			fmt.Printf("Password for %s: %s\n", result, string(secret.Data["password"]))

		}
		return nil
	},
}
