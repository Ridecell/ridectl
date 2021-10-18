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

	"github.com/Ridecell/ridecell-controllers/apis/db/v1beta2"
	"github.com/Ridecell/ridectl/pkg/kubernetes"
	"github.com/Ridecell/ridectl/pkg/utils"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	rootCmd.AddCommand(postgresdumpCMD)
}

var postgresdumpCMD = &cobra.Command{
	Use:   "postgresdump [flags] <microservice_name> <backup_name>",
	Short: "Take postgres DB dump",
	Long:  `Take postgres DB dump`,
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			pterm.Error.Printf("Microservice name argument is required.")
			os.Exit(1)
		}
		if len(args) == 1 {
			pterm.Error.Printf("Backup name argument is required.")
			os.Exit(1)
		}
		if len(args) > 2 {
			return fmt.Errorf("too many arguments")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		kubeconfig := utils.GetKubeconfig()
		target, err := kubernetes.ParseSubject(args[0])
		if err != nil {
			pterm.Error.Println(err, "Its not a valid Microservice")
			os.Exit(1)
		}
        
		kubeObj := kubernetes.GetAppropriateObjectWithContext(*kubeconfig, args[0], target)
		if reflect.DeepEqual(kubeObj, kubernetes.Kubeobject{}) {
			pterm.Error.Printf("No instance found %s\n", args[0])
			os.Exit(1)
		}
        
		//get users list to get postgresdatabase ref 
		postgresUserList := &v1beta2.PostgresUserList{}
		_ = kubeObj.Client.List(ctx, postgresUserList, client.InNamespace(target.Namespace))
		if len(postgresUserList.Items) == 0 {
			return errors.Wrap(err, "failed to get po users list")
		}
		postgresUser := &v1beta2.PostgresUser{}
		for _, postgresUsr := range postgresUserList.Items {
			if postgresUsr.Spec.Mode == "owner" {
				postgresUser = &postgresUsr
				break
			}
		}
		if postgresUser.Name == "" {
			return errors.Wrap(err, "failed to get postgres user")
		}
        
		// postgresdumpObj:=v1beta2.PostgresDump{
		// 	ObjectMeta: metav1.ObjectMeta{
		// 		Name:      args[1],
		// 	    Namespace: target.Namespace,
		// 	},
		// 	Spec: v1beta2.PostgresDumpSpec{
		// 		PostgresDatabaseRef: postgresUser.Spec.PostgresDatabaseRef,
		// 	},
		// }
		// err = kubeObj.Client.Create(ctx, postgresdumpObj)
		// if err != nil {
		// 	return errors.Wrap(err, "failed to create postgresdump obj")
		// }

		//TODO 
		//1] more informative error messages and print statements
		//2] package import error (merge in main)
		
		pterm.Success.Printf("Taking  postgres dump")
		return nil
	},
}
