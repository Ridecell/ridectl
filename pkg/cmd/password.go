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
	"strings"

	"github.com/Ridecell/ridectl/pkg/utils"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

func init() {
	rootCmd.AddCommand(passwordCmd)
}

var passwordCmd = &cobra.Command{
	Use:   "password [flags] <tenant_name>",
	Short: "Gets dispatcher/postgres readonly user password/connection details for a Summon Instance",
	Long:  "Returns dispatcher django password from a Summon Instance Secret or postgres connection details for readonly user\n" +
		"For summon instances: password <tenant>-<env>                   -- e.g. ridectl password darwin-qa\n" +
		"For microservices: password svc-<region>-<env>-<microservice>   -- e.g. ridectl password svc-us-master-dispatch",
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("tenant name argument is required")
		}
		if len(args) > 1 {
			return fmt.Errorf("too many arguments")
		}
		return nil
	},

	PreRunE: func(cmd *cobra.Command, args []string) error {
		utils.CheckTshLogin()
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		target, kubeObj, exist := utils.DoesInstanceExist(args[0], inCluster)

		if !exist {
			os.Exit(1)
		}

		// Defaults to postgresql in case of microservices
		secretType := "postgresql"

		var err error

		if target.Type == "summon" {
			secretTypes := []string{"django", "postgresql"}

			secretPrompt := promptui.Select{
				Label: "Select secret",
				Items: secretTypes,
			}

			_, secretType, err = secretPrompt.Run()
			if err != nil {
				return errors.Wrapf(err, "Prompt failed")
			}
		}

		secret := &corev1.Secret{}

		switch secretType {

		case "django":
			err = kubeObj.Client.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s.django-password", args[0]), Namespace: target.Namespace}, secret)
			if err != nil {
				return errors.Wrapf(err, "error getting secret for instance %s", args[0])
			}

			pterm.Success.Printf("Password for %s: %s\n", args[0], string(secret.Data["password"]))
			pterm.Warning.Printf("If someone has changed or reset the password manually, then above password will not work.\n")

		case "postgresql":
			// get a list of secrets which have readonly in their name
			readOnlysecrets := []string{}
			secrets := &corev1.SecretList{}

			err = kubeObj.Client.List(ctx, secrets, &client.ListOptions{
				Namespace: target.Namespace,
			})
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
				return errors.Errorf("no readonly secrets found for instance %s", args[0])
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
				return errors.Wrapf(err, "error getting secret for instance %s", args[0])
			}

			pterm.Success.Printf("Readonly User Connection Details\n")
			pterm.Success.Prefix = pterm.Prefix{
				Text: "",
			}
			pterm.Success.Printf("Database Type: Postgres\n") // Hard code-y
			pterm.Success.Printf("Database Host: %s\n", string(secret.Data["host"]))
			pterm.Success.Printf("Database Port: %s\n", string(secret.Data["port"]))
			pterm.Success.Printf("Database Name: %s\n", string(secret.Data["dbname"]))
			pterm.Success.Printf("Database Username: %s\n", string(secret.Data["username"]))
			pterm.Success.Printf("Database Password: %s\n", string(secret.Data["password"]))

		}

		return nil
	},
}
