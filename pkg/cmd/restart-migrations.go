/*
Copyright 2019-2020 Ridecell, Inc.

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

	"github.com/Ridecell/ridectl/pkg/kubernetes"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	batchv1 "k8s.io/api/batch/v1"
)

func init() {
	rootCmd.AddCommand(redeployCmd)
}

var redeployCmd = &cobra.Command{
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
	RunE: func(_ *cobra.Command, args []string) error {
		namespace := kubernetes.ParseNamespace(args[0])
		fetchObject := &kubernetes.KubeObject{
			Top: &batchv1.Job{},
		}
		err := kubernetes.GetObject(kubeconfigFlag, fmt.Sprintf("%s-migrations", args[0]), namespace, fetchObject)
		if err != nil {
			return errors.Wrap(err, "unable to find job")
		}

		job, ok := fetchObject.Top.(*batchv1.Job)
		if !ok {
			return errors.New("unable to convert runtime.object to batchv1.Job")
		}

		fmt.Printf("Restarting migrations for %s\n", args[0])

		err = fetchObject.Client.Delete(context.Background(), job)
		if err != nil {
			return err
		}

		return nil
	},
}
