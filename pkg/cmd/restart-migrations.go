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

	"github.com/Ridecell/ridectl/pkg/kubernetes"
	"github.com/Ridecell/ridectl/pkg/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	rootCmd.AddCommand(restartMigrationsCmd)
}

var restartMigrationsCmd = &cobra.Command{
	Use:   "restart-migrations [flags] <cluster_name> ",
	Short: "Restart migrations for target summon instance.",
	Long:  "Restart migrations for target summon instance.",
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("Cluster name argument is required")
		}
		if len(args) > 1 {
			return fmt.Errorf("Too many arguments")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		kubeconfig := utils.GetKubeconfig()
		target, err := kubernetes.ParseSubject(args[0])
		if err != nil {
			return errors.Wrapf(err, "not a valid target %s", args[0])
		}

		kubeObj := kubernetes.GetAppropriateObjectWithContext(*kubeconfig, args[0], target)
		if reflect.DeepEqual(kubeObj, kubernetes.Kubeobject{}) {
			return errors.Wrapf(err, "no instance found %s", args[0])
		}

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

		return nil
	},
}
