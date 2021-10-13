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
	"reflect"
	"time"

	"github.com/Ridecell/ridectl/pkg/kubernetes"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	utils "github.com/Ridecell/ridectl/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
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
	Use:   "restart [flags] <cluster_name> <pod_type>",
	Short: "Performs a rolling restart of pods.",
	Long: "Restarts all pods of a certain type (web|celeryd|etc).\n" +
		"For summon instances: restart <tenant>-<env> <type>                 -- e.g. ridectl restart summontest-dev web\n" +
		"For microservices: restart svc-<region>-<env>-<microservice> <type>  -- e.g. ridectl svc-us-master-webhook-sms web",
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("Cluster name argument is required.")
		}
		if len(args) == 1 {
			return fmt.Errorf("Deployment type argument is required.")
		}
		if len(args) > 2 {
			return fmt.Errorf("Too many arguments")
		}
		return nil
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		utils.CheckVPN()

		binaryExists := utils.CheckBinary("kubectl")
		if !binaryExists {
			return fmt.Errorf("kubectl is not installed. Follow the instructions here: https://kubernetes.io/docs/tasks/tools/#kubectl to install it")
		}
		fmt.Printf("\nWarning: This might cause downtime for your services\n")
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		kubeconfig := utils.GetKubeconfig()
		target, err := kubernetes.ParseSubject(args[0])
		if err != nil {
			return errors.Wrapf(err, "not a valid target %s", args[0])
		}

		var deploymentName string
		podLabels := make(map[string]string)

		if target.Type == "summon" {
			podLabels["app.kubernetes.io/instance"] = fmt.Sprintf("%s-web", args[0])
			deploymentName = fmt.Sprintf("%s-%s", args[0], args[1])
		} else if target.Type == "microservice" {
			podLabels["app"] = fmt.Sprintf("%s-svc-%s", target.Env, target.Namespace)
			podLabels["environment"] = target.Env
			podLabels["region"] = target.Region
			podLabels["role"] = args[1]
			deploymentName = fmt.Sprintf("%s-svc-%s-%s", target.Env, target.Namespace, args[1])
		}

		kubeObj := kubernetes.GetAppropriateObjectWithContext(*kubeconfig, args[0], target)
		if reflect.DeepEqual(kubeObj, kubernetes.Kubeobject{}) {
			return errors.Wrapf(err, "no instance found %s", args[0])
		}

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

		var restartSuccess bool
		for !restartSuccess {
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
				err = wait.Poll(time.Second*5, time.Minute*3, func() (bool, error) {
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
			restartSuccess = true
		}
		pterm.Success.Printf("Successfully restarted pods for %s : %s\n", args[0], args[1])
		return nil
	},
}
