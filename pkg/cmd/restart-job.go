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

	"github.com/Ridecell/ridectl/pkg/kubernetes"
	"github.com/Ridecell/ridectl/pkg/utils"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	rootCmd.AddCommand(restartJobCmd)
}

var restartJobCmd = &cobra.Command{
	Use:   "restart-job",
	Short: "Restart migration or postgresDump job",
	Long: "Restart migration job for target summon instance OR restart postgresDump failed job.\n" +
		"restart-job, select jobtype and enter required input as per instruction.\n"+
		"eg.ridectl restart-job -> Select job type: PostgreDump -> Enter postgresDump name = enter_postgresDump_name",
	Args: func(_ *cobra.Command, args []string) error {
		return nil
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		utils.CheckVPN()
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		jobTypes := []string{"Migration", "PostgresDump"}
		jobPromt := promptui.Select{
			Label: "Select job type ",
			Items: jobTypes,
		}
		_, jobType, err := jobPromt.Run()
		if err != nil {
			return errors.Wrapf(err, "Prompt failed")
		}
		validator := func(input string) error {
			if input == "" {
				return errors.New("Invalid summont tenant/ microservice name or postgresDump name")
			}
			return nil
		}
		instanceNamePromt := promptui.Prompt{
			Label:    "Enter summon tenant(sandbox-dev)/microservice(svc-us-master-microservice) name",
			Validate: validator,
		}
		if jobType == "PostgresDump" {
			instanceNamePromt.Label = "Enter postgresDump name"
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

		if jobType == "Migration" {
			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-migrations", target.Name),
					Namespace: target.Namespace},
			}
			// deleting the migrations job restarts the migrations
			err = kubeObj.Client.Delete(ctx, job)
			if err != nil {
				return errors.Wrap(err, "failed to restart job")
			}

			pterm.Success.Printf("Restarted migrations for %s\n", target.Name)
		} else {
			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name+"-pgdump",
					Namespace: target.Namespace},
			}
			
			err = kubeObj.Client.Delete(ctx, job)
			if err != nil {
				return errors.Wrap(err, "failed to restart job")
			}

			pterm.Success.Printf("Restarted postgresDump for %s\n", target.Name)
		}
		return nil
	},
}
