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
	"strings"
	"time"

	"github.com/Ridecell/ridectl/pkg/kubernetes"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	utils "github.com/Ridecell/ridectl/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	rootCmd.AddCommand(rollingRestartCmd)
}

/*
We are using summon-operator to create deployments of summon-platform. So when we try to do rollout restart,
it does not behave as expected because summon-operator is constantly watching the deployments and reconciles if anything changes.
So when we do rollout restart k8s starts a new pod but it gets terminated immediately because summon-operator restarts.
This becomes very tricky to handle in operator-itself. Hence we have implemented another way to do this
Ref: https://ridecell.atlassian.net/browse/DEVOPS-2925
*/
var rollingRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Performs restart of Pods, migration job or Postgresdump job.",
	Long: "Restarts pods or jobs depending on user's selection.\n\n" +
		"Specify instance name / microservice name in following format:\n" +
		"Summon instances :   <tenant>-<env>                      -- e.g. summontest-dev\n" +
		"Microservices    :   svc-<region>-<env>-<microservice>   -- e.g. svc-us-master-webhook-sms\n\n" +
		"For restarting pods, provide component name. For example:\n" +
		"  Summon components: web, celeryd, static, celeryredbeat, kafkaconsumer, daphne, channelworker, platform-one, etc\n" +
		"  Microservice components: web, celery-beat, celery-worker, kafka-consumer, etc",
	Args: func(_ *cobra.Command, args []string) error {
		return nil
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		utils.CheckTshLogin()
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		validateInstance := func(input string) error {
			if strings.Contains(input, " ") {
				return errors.New("Remove white-spaces from input [" + input + "]")
			}
			_, err := kubernetes.ParseSubject(input)
			if err != nil {
				return errors.New("Its not a valid Summonplatform or Microservice")
			}
			return nil
		}
		validateInput := func(input string) error {
			if input == "" {
				return errors.New("Invalid component or Postgresdump object name")
			}
			if strings.Contains(input, " ") {
				return errors.New("Remove white-spaces from input [" + input + "]")
			}
			return nil
		}

		restartTypes := []string{"Migration", "Pods", "PostgresDump Job"}

		restartPrompt := promptui.Select{
			Label: "Select what to restart:",
			Items: restartTypes,
		}

		_, restartType, err := restartPrompt.Run()
		if err != nil {
			return errors.Wrapf(err, "Prompt failed")
		}

		switch restartType {
		case "Migration":
			prompt := promptui.Prompt{
				Label:    "Enter SummonPlatform instance name (e.g. darwin-qa)",
				Validate: validateInstance,
			}
			instanceName, err := prompt.Run()
			if err != nil {
				return errors.Wrapf(err, "Prompt failed")
			}

			target, kubeObj, exist := utils.DoesInstanceExist(instanceName, inCluster)

			if !exist {
				os.Exit(1)
			}

			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-migrations", target.Name),
					Namespace: target.Namespace},
			}
			// deleting the migrations job restarts the migrations
			err = kubeObj.Client.Delete(context.TODO(), job)
			if err != nil {
				return errors.Wrap(err, "failed to restart job")
			}
			pterm.Success.Printf("Restarted migrations for %s\n", target.Name)

		case "Pods":
			pterm.Warning.Println("Warning: This might cause downtime for your services.")

			prompt := promptui.Prompt{
				Label:    "Enter SummonPlatform/Microservice name (e.g. darwin-qa or svc-us-master-webhook-sms)",
				Validate: validateInstance,
			}
			instanceName, err := prompt.Run()
			if err != nil {
				return errors.Wrapf(err, "Prompt failed")
			}
			prompt = promptui.Prompt{
				Label:    "Enter component type (e.g. web, celeryd/celery-worker, static, celeryredbeat/celery-beat, kafkaconsumer/kafka-consumer, etc)",
				Validate: validateInput,
			}
			component, err := prompt.Run()
			if err != nil {
				return errors.Wrapf(err, "Prompt failed")
			}

			target, kubeObj, exist := utils.DoesInstanceExist(instanceName, inCluster)
			if !exist {
				os.Exit(1)
			}

			var deploymentName string
			podLabels := make(map[string]string)

			if target.Type == "summon" {
				podLabels["app.kubernetes.io/instance"] = fmt.Sprintf("%s-%s", target.Name, component)
				deploymentName = fmt.Sprintf("%s-%s", target.Name, component)
			} else if target.Type == "microservice" {
				podLabels["app"] = fmt.Sprintf("%s-svc-%s", target.Env, target.Namespace)
				podLabels["environment"] = target.Env
				podLabels["region"] = target.Region
				podLabels["role"] = component
				deploymentName = fmt.Sprintf("%s-svc-%s-%s", target.Env, target.Namespace, component)
			}

			pterm.Info.Printf("Restarting pods for %s : %s\n", target.Name, component)

			labelSet := labels.Set{}
			for k, v := range podLabels {
				labelSet[k] = v
			}
			listOptions := &client.ListOptions{
				Namespace:     target.Namespace,
				LabelSelector: labels.SelectorFromSet(labelSet),
			}
			pods := &corev1.PodList{}
			err = kubeObj.Client.List(context.TODO(), pods, listOptions)
			if err != nil {
				return errors.Wrap(err, "error listing pods")
			}

			deployment := &appsv1.Deployment{}

			for _, pod := range pods.Items {
				err = kubeObj.Client.Delete(context.TODO(), &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      pod.Name,
						Namespace: target.Namespace,
					},
				})
				if err != nil {
					return errors.Wrap(err, "error deleting pod")
				}

				// waiting for the deployment to be ready
				err = wait.PollUntilContextTimeout(context.TODO(), time.Second*5, time.Minute*3, false, func(ctx context.Context) (bool, error) {
					_ = kubeObj.Client.Get(context.TODO(), types.NamespacedName{Name: deploymentName, Namespace: target.Namespace}, deployment)

					if deployment.Status.ReadyReplicas == deployment.Status.Replicas {
						return true, nil
					}
					return false, nil
				})
				if err != nil {
					return errors.Wrap(err, "error waiting for deployment to re ready")
				}
			}
			pterm.Success.Printf("Successfully restarted pods for %s : %s\n", target.Name, component)

		case "PostgresDump Job":
			prompt := promptui.Prompt{
				Label:    "Enter Postgresdump object name",
				Validate: validateInput,
			}
			pgdumpName, err := prompt.Run()
			if err != nil {
				return errors.Wrapf(err, "Prompt failed")
			}
			prompt = promptui.Prompt{
				Label:    "Enter Postgresdump object namespace",
				Validate: validateInput,
			}
			pgdumpNamespace, err := prompt.Run()
			if err != nil {
				return errors.Wrapf(err, "Prompt failed")
			}

			// Create Subject object for PostgresDump Job object
			target := kubernetes.Subject{
				Name:      pgdumpName + "-pgdump",
				Namespace: pgdumpNamespace,
				Type:      "job",
			}

			kubeconfig := utils.GetKubeconfig()
			kubeObj, err := kubernetes.GetAppropriateObjectWithContext(*kubeconfig, "", target, inCluster)
			if err != nil {
				pterm.Error.Printf("%s", err.Error())
				os.Exit(1)
			}
			if reflect.DeepEqual(kubeObj, kubernetes.Kubeobject{}) {
				pterm.Error.Printf("No PostgresDump job found %s\n", pgdumpName+"-pgdump")
				os.Exit(1)
			}

			jobObj := kubeObj.Object.(*batchv1.Job)
			// deleting the postgresdump job restarts the postgresdump backup process
			err = kubeObj.Client.Delete(context.TODO(), jobObj)
			if err != nil {
				return errors.Wrap(err, "failed to restart job")
			}
			pterm.Success.Printf("Restarted PostgresDump job for %s\n", pgdumpName)
		}
		return nil
	},
}
