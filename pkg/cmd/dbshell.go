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
	//"strings"

	//"github.com/pkg/errors"
	"github.com/spf13/cobra"
	//"github.com/Ridecell/ridectl/pkg/exec"
	//"github.com/Ridecell/ridectl/pkg/kubernetes"
)

func init() {
	rootCmd.AddCommand(dbShellCmd)
}

var dbShellCmd = &cobra.Command{
	Use:   "dbshell [flags] <cluster_name>",
	Short: "Open a database shell on a Summon instance",
	Long:  `Open an interactive PostgreSQL shell for a Summon instance running on Kubernetes`,
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
		fmt.Println("This feature is currently disabled.")
		return nil

		//namespace := strings.Split(args[0], "-")[1]
		//
		//clientset, err := kubernetes.GetClient(kubeconfigFlag)
		//if err != nil {
		//	return errors.Wrap(err, "unable to load Kubernetes configuration")
		//}
		//
		//// dynamicClient is needed to retrieve custom resources (for now)
		//dynamicClient, err := kubernetes.GetDynamicClient(kubeconfigFlag)
		//if err != nil {
		//	return err
		//}
		//// Retrieve our SummonPlatform object for specified cluster
		//summonObject, err := kubernetes.FindSummonObject(dynamicClient, args[0])
		//if err != nil {
		//	return err
		//}
		//
		//// Check if datbaseSpec exists in our summon object
		//var exclusiveDatabase bool
		//var sharedDatabaseName string
		//// Messy hack to grab just the things we care about
		//databaseSpec, ok := summonObject.Object["spec"].(map[string]interface{})["database"].(map[string]interface{})
		//if ok {
		//	exclusiveDatabase, ok = databaseSpec["exclusiveDatabase"].(bool)
		//	if !ok {
		//		exclusiveDatabase = false
		//	}
		//	sharedDatabaseName, ok = databaseSpec["sharedDatabaseName"].(string)
		//	if !ok {
		//		sharedDatabaseName = namespace
		//	}
		//} else {
		//	exclusiveDatabase = false
		//	sharedDatabaseName = namespace
		//}
		//
		//if exclusiveDatabase {
		//	pod, err := kubernetes.FindSummonPod(clientset, args[0], "application=spilo,spilo-role=master")
		//	if err != nil {
		//		return errors.Wrap(err, "unable to find pod")
		//	}
		//	fmt.Printf("Connecting to %s/%s\n", pod.Namespace, pod.Name)
		//	// Spawn kubectl exec.
		//	kubectlArgs := []string{"kubectl", "exec", "-it", "-n", pod.Namespace, pod.Name, "--", "bash", "-c", "psql -U summon summon"}
		//	return exec.Exec(kubectlArgs)
		//}
		//
		//// If we got here we're using a shared database
		//pod, err := kubernetes.GetPod(clientset, namespace, fmt.Sprintf(`^%s-database-[0-9]+`, sharedDatabaseName), "application=spilo,spilo-role=master")
		//if err != nil {
		//	return errors.Wrap(err, "unable to find pod")
		//}
		//dbUsername := strings.Replace(args[0], "-", "_", -1)
		//kubectlArgs := []string{"kubectl", "exec", "-it", "-n", pod.Namespace, pod.Name, "--", "bash", "-c", fmt.Sprintf("psql -U %s", dbUsername)}
		//return exec.Exec(kubectlArgs)
	},
}
