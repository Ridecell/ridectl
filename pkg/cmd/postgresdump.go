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
	osExec "os/exec"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Ridecell/ridecell-controllers/apis/db/v1beta2"
	"github.com/Ridecell/ridectl/pkg/kubernetes"
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

var check bool

func init() {
	postgresdumpCMD.Flags().BoolVarP(&check, "check", "f", false, "(optional) follows the status of postgresdump instance until terminated")
}

func getInstanceData(objName string, context string, namespace string) (string, error) {
	var data []byte
	var err error
	instanceData, err := TempFS.ReadFile("templates/show_postgresdump.tpl")
	if err != nil {
		return "", errors.Wrap(err, "error reading show_postgresdump.tpl")
	}

	data, err = osExec.Command("kubectl", "get", "postgresdumps.db.controllers.ridecell.io", objName, "-n", namespace, "--context", context, "-o", "go-template="+string(instanceData)).Output()
	if err != nil {
		return "", errors.Wrap(err, "error getting postgresdump instance info")
	}

	return string(data), err
}

var postgresdumpCMD = &cobra.Command{
	Use:   "postgresdump [flags] <microservice_name> <backup_name>",
	Short: "Take postgres DB dump",
	Long:  `Take postgres DB dump`,
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("microservice name argument is required")
		}
		if len(args) > 2 {
			return fmt.Errorf("too many arguments")
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

		postgresUserList := &v1beta2.PostgresUserList{}
		_ = kubeObj.Client.List(ctx, postgresUserList, client.InNamespace(target.Namespace))
		if len(postgresUserList.Items) == 0 {
			return errors.Wrap(err, "failed to get postgres users list")
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
		pterm.Success.Printf("Taking postgres dump\n")
		data, err := getInstanceData(instanceName, kubeObj.Context.Cluster, target.Namespace)
		if err != nil {
			return err
		}
		if check {
			count := 0
			pterm.Info.Printf("Updating status")
			pterm.Printf("\n")
			area, _ := pterm.DefaultArea.Start()
			for {
				status := pterm.FgLightMagenta.Sprint(data)
				area.Update(status)
				if strings.Contains(data, "STATUS: Completed") {
					pterm.Success.Printf("Done!!")
					break
				}
				if strings.Contains(data, "STATUS: Error") {
					pterm.Error.Printf("Error!!")
					break
				}
				count++
				if count > 300 {
					pterm.Error.Printf("Error!!")
					break
				}
				time.Sleep(time.Second * 2)
				data, err = getInstanceData(instanceName, kubeObj.Context.Cluster, target.Namespace)
				if err != nil {
					return err
				}
			}
			err = area.Stop()
			if err != nil {
				return err
			}
		} else {
			pterm.Success.Printf(data)
		}
		return nil
	},
}
