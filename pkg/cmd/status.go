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
	"os"
	osExec "os/exec"
	"time"

	"github.com/apoorvam/goterminal"
	"github.com/pkg/errors"
	"github.com/shurcooL/httpfs/vfsutil"
	"github.com/spf13/cobra"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridectl/pkg/kubernetes"
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
		summonData, err := vfsutil.ReadFile(Templates, "show_summon.tpl")
		if err != nil {
			return "", errors.Wrap(err, "error reading show_summon.tpl")
		}

		data, err = osExec.Command("kubectl", "get", "summon", "-n", namespace, "--context", context, tenant, "-o", "go-template="+string(summonData)).Output()
	} else if objType == "deployment" {
		deploymentData, err := vfsutil.ReadFile(Templates, "show_deployments.tpl")
		if err != nil {
			return "", errors.Wrap(err, "error reading show_deployments.tpl")
		}

		data, err = osExec.Command("kubectl", "get", "deployment", "-n", namespace, "--context", context, "-l", "app.kubernetes.io/part-of="+tenant, "-o", "go-template="+string(deploymentData)).Output()
	}

	return string(data), err
}

var statusCmd = &cobra.Command{
	Use:   "status [follow] <cluster_name>",
	Short: "Get status report of an Summon Instance",
	Long:  "Shows status details for all components of a Summon Instance",
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
		target, err := kubernetes.ParseSubject(args[0])
		if err != nil {
			return errors.Wrap(err, "not a valid target")
		}

		fetchObject := &kubernetes.KubeObject{Top: &summonv1beta1.SummonPlatform{}}
		err = kubernetes.GetObject(kubeconfigFlag, target.Name, target.Namespace, fetchObject)
		if err != nil {
			return err
		}

		sData, err := getData("summon", fetchObject.Context.Name, target.Namespace, args[0])
		if err != nil {
			return err
		}

		dData, err := getData("deployment", fetchObject.Context.Name, target.Namespace, args[0])
		if err != nil {
			return err
		}

		if follow {
			writer := goterminal.New(os.Stdout)
			indicator := goterminal.New(os.Stdout)
			spinner := []string{"|", "/", "-", "\\", "/"}
			steps := 0
			for true {
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
			}
		} else {
			fmt.Printf(sData + "\n" + dData + "\n")
		}

		return nil
	},
}
