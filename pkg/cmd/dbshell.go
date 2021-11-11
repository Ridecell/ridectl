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

	"github.com/Ridecell/ridectl/pkg/exec"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	kubernetes "github.com/Ridecell/ridectl/pkg/kubernetes"
	utils "github.com/Ridecell/ridectl/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func init() {
	rootCmd.AddCommand(dbShellCmd)
}

var dbShellCmd = &cobra.Command{
	Use:   "dbshell [flags] <cluster_name>",
	Short: "Open a database shell on a Summon instance or microservice",
	Long: "Open an interactive PostgreSQL shell for a Summon instance or microservice running on Kubernetes.\n" +
		"For summon instances: dbshell <tenant>-<env>                   -- e.g. ridectl dbshell darwin-qa\n" +
		"For microservices: dbshell svc-<region>-<env>-<microservice>   -- e.g. ridectl dbshell svc-us-master-dispatch",
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
		utils.CheckPsql()
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		kubeconfig := utils.GetKubeconfig()
		target, err := kubernetes.ParseSubject(args[0])
		if err != nil {
			pterm.Error.Println(err, "Its not a valid Summonplatform or Microservice")
			os.Exit(1)
		}
		kubeObj := kubernetes.GetAppropriateObjectWithContext(*kubeconfig, args[0], target, inCluster)
		if reflect.DeepEqual(kubeObj, kubernetes.Kubeobject{}) {
			pterm.Error.Printf("No instance found %s\n", args[0])
			os.Exit(1)
		}
		secretObj := &corev1.Secret{}
		err = kubeObj.Client.Get(context.Background(), types.NamespacedName{Name: target.Name + ".postgres-user-password", Namespace: target.Namespace}, secretObj)
		if err != nil {
			pterm.Error.Printf("instance not found in %s", kubeObj.Context.Cluster)
			os.Exit(1)
		}

		psqlCmd := []string{"psql", "-h", string(secretObj.Data["host"]), "-U", string(secretObj.Data["username"]), string(secretObj.Data["dbname"])}
		os.Setenv("PGPASSWORD", string(secretObj.Data["password"]))
		return exec.Exec(psqlCmd)
	},
}
