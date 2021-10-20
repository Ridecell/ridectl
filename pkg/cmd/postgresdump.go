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
	"github.com/apoorvam/goterminal"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	rootCmd.AddCommand(postgresdumpCMD)
}

var check bool

func init() {
	statusCmd.Flags().BoolVarP(&check, "check", "-f", false, "(optional) follows the status of postgresdump instance until terminated")
}

func getInstanceData(objName string, context string, namespace string) (string, error) {
	var data []byte
	var err error
	instanceData, err := TempFS.ReadFile("template/show_postgresdump.tpl")
	if err != nil {
		return "", errors.Wrap(err, "error reading show_summon.tpl")
	}

	data, err = osExec.Command("kubectl", "get", "postgresdumps.db.controllers.ridecell.io", objName, "-n", namespace, "--context", context, "-o", "go-template="+string(instanceData)).Output()
	if err != nil {
		return "", errors.Wrap(err, err.Error())
	}

	return string(data), err
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
            instanceName= args[1]
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
			if k8serror.IsAlreadyExists(err) {
				postgresdumpObj.Name = args[0] + "-" + strconv.FormatInt(time.Now().Unix(), 10)
				err = kubeObj.Client.Create(ctx, postgresdumpObj)
				if err != nil {
					return errors.Wrap(err, "failed to create postgresdump instance")
				}
			}
			return errors.Wrap(err, "failed to create postgresdump instance")
		}
		pterm.Success.Printf("Taking postgres dump")

		data, err := getInstanceData(args[1], kubeObj.Context.Cluster, target.Namespace)
		if err != nil {
			return err
		}
		if check {
			writer := goterminal.New(os.Stdout)
			indicator := goterminal.New(os.Stdout)
			spinner := []string{"|", "/", "-", "\\", "/"}
			steps := 0
			for {
				writer.Clear()
				fmt.Fprintf(writer, "%s\n", data)
				writer.Print()
				for i := 0; i < 3; i++ {
					fmt.Fprintf(indicator, "%s\n", spinner[steps%len(spinner)])
					indicator.Print()
					time.Sleep(time.Second)
					indicator.Clear()
					steps++
				}
				if strings.Contains(data, "STATUS: Succeeded") {
					pterm.Success.Printf("Done!!")
					break
				}
				if strings.Contains(data, "STATUS: Error") {
					pterm.Error.Printf("Error!!")
					break
				}
				data, err = getInstanceData(args[1], kubeObj.Context.Cluster, target.Namespace)
				if err != nil {
					return err
				}
			}
		} else {
			pterm.Success.Printf(data)
		}
		return nil
	},
}
