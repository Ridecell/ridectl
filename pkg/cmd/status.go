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
	"os"
	osExec "os/exec"
	"reflect"
	"time"

	"github.com/Ridecell/ridectl/pkg/utils"
	"github.com/apoorvam/goterminal"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	kubernetes "github.com/Ridecell/ridectl/pkg/kubernetes"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var follow bool

func init() {
	statusCmd.Flags().BoolVarP(&follow, "follow", "f", false, "(optional) follows the status of tenant until terminated")
}

// Helper functions for running kubectl commands to retrieve object info.
func getData(objType string, context string, namespace string, tenant string) (string, error) {
	var data []byte
	var err error

	if objType == "summon" {
		summonData, err := TempFS.ReadFile("templates/show_summon.tpl")
		if err != nil {
			return "", errors.Wrap(err, "error reading show_summon.tpl")
		}

		data, err = osExec.Command("kubectl", "get", "summonplatform.app.summon.ridecell.io", "-n", namespace, "--context", context, tenant, "-o", "go-template="+string(summonData)).Output()
		if err != nil {
			return "", errors.Wrap(err, "error getting summon platform info")
		}
	} else if objType == "deployment" {
		deploymentData, err := TempFS.ReadFile("templates/show_deployments.tpl")
		if err != nil {
			return "", errors.Wrap(err, "error reading show_deployments.tpl")
		}

		data, err = osExec.Command("kubectl", "get", "deployment", "-n", namespace, "--context", context, "-l", "app.kubernetes.io/part-of="+tenant, "-o", "go-template="+string(deploymentData)).Output()
		if err != nil {
			return "", errors.Wrap(err, "error getting deployment info")
		}
	}

	return string(data), err
}

var statusCmd = &cobra.Command{
	Use:   "status [follow] <cluster_name>",
	Short: "Get status report of an Summon Instance",
	Long:  "Shows status details for all components of a Summon Instance",
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			pterm.Error.Println("cluster name argument is required")
			os.Exit(1)
		}
		if len(args) > 1 {
			pterm.Error.Println("too many arguments")
			os.Exit(1)
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
	RunE: func(_ *cobra.Command, args []string) error {
		kubeconfig := utils.GetKubeconfig()
		target, err := kubernetes.ParseSubject(args[0])
		if err != nil {
			pterm.Error.Println(err, "Its not a valid target")
			os.Exit(1)
		}

		kubeObj := kubernetes.GetAppropriateObjectWithContext(*kubeconfig, args[0], target)
		if reflect.DeepEqual(kubeObj, kubernetes.Kubeobject{}) {
			pterm.Error.Printf("No instance found %s\n", args[0])
			os.Exit(1)
		}

		sData, err := getData("summon", kubeObj.Context.Cluster, target.Namespace, args[0])
		if err != nil {
			return err
		}
		dData, err := getData("deployment", kubeObj.Context.Cluster, target.Namespace, args[0])
		if err != nil {
			return err
		}
		if follow {
			writer := goterminal.New(os.Stdout)
			indicator := goterminal.New(os.Stdout)
			spinner := []string{"|", "/", "-", "\\", "/"}
			steps := 0
			for {
				writer.Clear()
				fmt.Fprintf(writer, "%s\n%s\n", sData, dData)
				writer.Print()
				// Wait 3 seconds before next command run and display. Also show progression
				// with spinner steps. Note that goterminal seems to rely on new lines in order
				// to appear as desired
				for i := 0; i < 3; i++ {
					fmt.Fprintf(indicator, "%s\n", spinner[steps%len(spinner)])
					indicator.Print()
					time.Sleep(time.Second)
					indicator.Clear()
					steps++
				}
				// Calling it at end of for loop since we made these calls right before.
				sData, err = getData("summon", kubeObj.Context.Cluster, target.Namespace, args[0])
				if err != nil {
					return err
				}

				dData, err = getData("deployment", kubeObj.Context.Cluster, target.Namespace, args[0])
				if err != nil {
					return err
				}
			}
		} else {
			pterm.Success.Printf(sData + "\n" + dData + "\n")
		}
		return nil
	},
}
