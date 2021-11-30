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
	"os"
	osExec "os/exec"
	"reflect"
	"time"

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
	} else {
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
	Use:   "status",
	Short: "Get status report of an Summon Instance",
	Long:  "Shows status details for all components of a Summon Instance",
	Args: func(_ *cobra.Command, args []string) error {
		return nil
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		utils.CheckVPN()
		utils.CheckKubectl()
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		statusTypes := []string{"Summon Platform Status", "DB Backup Status"}
		statusPrompt := promptui.Select{
			Label: "Select ",
			Items: statusTypes,
		}
		_, statusType, err := statusPrompt.Run()
		if err != nil {
			return errors.Wrapf(err, "Prompt failed")
		}
		validator := func(input string) error {
			if input == "" {
				return errors.New("Invalid summont tenant name or microservice name")
			}
			return nil
		}
		instanceNamePromt := promptui.Prompt{
			Label:    "Enter summon tenant(sandbox-dev)/microservice(svc-us-master-microservice) name",
			Validate: validator,
		}
		name, err := instanceNamePromt.Run()
		if err != nil {
			return errors.Wrapf(err, "Prompt failed")
		}

		kubeconfig := utils.GetKubeconfig()
		target, err := kubernetes.ParseSubject(name)
		if err != nil {
			pterm.Error.Println(err, "Its not a valid Summonplatform or Microservice")
			os.Exit(1)
		}

		kubeObj := kubernetes.GetAppropriateObjectWithContext(*kubeconfig, name, target, inCluster)
		if reflect.DeepEqual(kubeObj, kubernetes.Kubeobject{}) {
			pterm.Error.Printf("No instance found %s\n", name)
			os.Exit(1)
		}

		var sData, dData, pData string
		if statusType == "Summon Platform Status" {
			sData, err = getData("summon", kubeObj.Context.Cluster, target.Namespace, name)
			if err != nil {
				return err
			}
			dData, err = getData("deployment", kubeObj.Context.Cluster, target.Namespace, name)
			if err != nil {
				return err
			}
		} else {
			pData, err = getData("postgresdump", kubeObj.Context.Cluster, target.Namespace, name)
			if err != nil {
				return err
			}
		}
		followStatus, _ := cmd.Flags().GetBool("follow")
		if followStatus {
			area, _ := pterm.DefaultArea.WithRemoveWhenDone().Start()
			for {
				p := pterm.DefaultProgressbar
				p.ShowElapsedTime = false
				p.RemoveWhenDone = true
				if statusType == "Summon Platform Status" {
					p = *p.WithTotal(2)
					area.Update(sData, "\n", dData)
					_, _ = p.Start()
					p.Title = "Fetching data"
					sData, err = getData("summon", kubeObj.Context.Cluster, target.Namespace, name)
					if err != nil {
						return err
					}
					p.Increment()

					dData, err = getData("deployment", kubeObj.Context.Cluster, target.Namespace, name)
					if err != nil {
						return err
					}
					p.Increment()
				} else {
					p = *p.WithTotal(1)
					area.Update(pData)
					_, _ = p.Start()
					p.Title = "Fetching data"
					pData, err = getData("posgtresdump", kubeObj.Context.Cluster, target.Namespace, name)
					if err != nil {
						return err
					}
					p.Increment()
					time.Sleep(time.Second * 10)
				}
				_, _ = p.Stop()
				_ = area.Stop()
			}
		} else {
			if statusType == "summonplatform" {
				pterm.Success.Printf(sData + "\n" + dData)
			} else {
				pterm.Success.Printf(pData)
			}
		}
		return nil
	},
}
