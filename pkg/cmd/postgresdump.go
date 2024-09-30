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
	"strconv"
	"strings"
	"time"

	"github.com/Ridecell/ridecell-controllers/apis/db/v1beta2"
	"github.com/Ridecell/ridectl/pkg/utils"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	rootCmd.AddCommand(postgresdumpCMD)
}

var postgresdumpCMD = &cobra.Command{
	Use:   "postgresdump [flags] <microservice_or_tenant_name> <backup_name>",
	Short: "Take postgres DB dump",
	Long:  `Take postgres DB dump, encrypt backup file and push it to s3 bucket`,
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("microservice or tenant name argument is required")
		}
		if len(args) > 2 {
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

		args[0] = strings.ToLower(args[0])
		if len(args) == 2 {
			args[1] = strings.ToLower(args[1])
		}

		target, kubeObj, exist := utils.DoesInstanceExist(args[0], inCluster, kubeconfigFlag)

		if !exist {
			os.Exit(1)
		}

		postgresUserList := &v1beta2.PostgresUserList{}
		err := kubeObj.Client.List(ctx, postgresUserList, client.InNamespace(target.Namespace))
		if err != nil {
			return errors.Wrap(err, "failed to get postgres user")
		}

		postgresUser := &v1beta2.PostgresUser{}
		for _, postgresUsr := range postgresUserList.Items {
			if postgresUsr.Spec.Mode == "owner" && postgresUsr.Name == args[0] {
				postgresUser = &postgresUsr
				break
			}
		}
		if postgresUser.Name == "" {
			return fmt.Errorf("No Postgres users of type owner found in namespace: %s", target.Namespace)
		}

		var instanceName string
		if len(args) < 2 {
			instanceName = args[0] + "-" + strconv.FormatInt(time.Now().Unix(), 10)
		} else {
			instanceName = args[1] + "-" + strconv.FormatInt(time.Now().Unix(), 10)
		}
		postgresdumpObj := &v1beta2.PostgresDump{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instanceName,
				Namespace: target.Namespace,
			},
			Spec: v1beta2.PostgresDumpSpec{
				PostgresDatabaseRef: postgresUser.Spec.PostgresDatabaseRef,
			},
		}
		err = kubeObj.Client.Create(ctx, postgresdumpObj)
		if err != nil {
			return errors.Wrap(err, "failed to create postgresdump instance")
		}
		// Do not change the following output format, kubernetes-microservices deploy workflow uses it in Backup DB step.
		pterm.Info.Printf("Created postgresdump kind with Name: %s Namespace: %s .\n", instanceName, postgresdumpObj.Namespace)
		pterm.Info.Printf("You can check status of DB backup using 'ridectl status' command \n")
		return nil
	},
}
