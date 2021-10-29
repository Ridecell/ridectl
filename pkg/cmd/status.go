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

	"github.com/Ridecell/ridectl/pkg/utils"
	"github.com/manifoldco/promptui"
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
	} else if objType == "postgresdump" {
		objectData, err := TempFS.ReadFile("templates/show_postgresdump.tpl")
		if err != nil {
			return "", errors.Wrap(err, "error reading show_postgresdump.tpl")
		}

		data, err = osExec.Command("kubectl", "get", "postgresdumps.db.controllers.ridecell.io", "-n", namespace, "--context", context, "-o", "go-template="+string(objectData)).Output()
		if err != nil {
			return "", errors.Wrap(err, "error getting postgresdump instance info")
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
			return fmt.Errorf("cluster name argument is required")
		}
		if len(args) > 1 {
			return fmt.Errorf("too many arguments")
		}
		return nil
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {

		utils.CheckVPN()
		utils.CheckKubectl()
		return nil
	},
	RunE: func(_ *cobra.Command, args []string) error {
		statusTypes := []string{"summonplatform", "postgresdump"}
		statusPrompt := promptui.Select{
			Label: "Select ",
			Items: statusTypes,
		}
		_, statusType, err := statusPrompt.Run()

		kubeconfig := utils.GetKubeconfig()
		target, err := kubernetes.ParseSubject(args[0])
		if err != nil {
			pterm.Error.Println(err, "Its not a valid Summonplatform or Microservice")
			os.Exit(1)
		}

		kubeObj := kubernetes.GetAppropriateObjectWithContext(*kubeconfig, args[0], target)
		if reflect.DeepEqual(kubeObj, kubernetes.Kubeobject{}) {
			pterm.Error.Printf("No instance found %s\n", args[0])
			os.Exit(1)
		}

		if err != nil {
			return errors.Wrapf(err, "Prompt failed")
		}
		var sData, dData, pData string
		if statusType == "summonplatform" {
			sData, err = getData("summon", kubeObj.Context.Cluster, target.Namespace, args[0])
			if err != nil {
				return err
			}
			dData, err = getData("deployment", kubeObj.Context.Cluster, target.Namespace, args[0])
			if err != nil {
				return err
			}
		}
		if statusType == "posgtresdump" {
			pData, err = getData("posgtresdump", kubeObj.Context.Cluster, target.Namespace, args[0])
			if err != nil {
				return err
			}
		}

		if follow {
			area, _ := pterm.DefaultArea.WithRemoveWhenDone().Start()

			for {
				if statusType == "summonplatform" {

					area.Update(sData, "\n", dData)
					sData, err = getData("summon", kubeObj.Context.Cluster, target.Namespace, args[0])
					if err != nil {
						return err
					}
					//area.Update(sData)

					dData, err = getData("deployment", kubeObj.Context.Cluster, target.Namespace, args[0])
					if err != nil {
						return err
					}

				} else {
					area.Update(pData)
					pData, err = getData("posgtresdump", kubeObj.Context.Cluster, target.Namespace, args[0])
					if err != nil {
						return err
					}
				}

				p := pterm.DefaultProgressbar.WithTotal(2)
				p.ShowElapsedTime = false
				p.RemoveWhenDone = true
				_, _ = p.Start()
				p.Title = "Fetching data"
				p.Increment()
				_, _ = p.Stop()
				_ = area.Stop()

			}
		} else {
			pterm.Success.Printf(sData + "\n" + dData + "\n" + "\n" + pData)
		}
		return nil
	},
}
