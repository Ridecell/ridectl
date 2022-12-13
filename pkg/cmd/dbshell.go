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
	"strings"

	"github.com/Ridecell/ridectl/pkg/exec"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"

	utils "github.com/Ridecell/ridectl/pkg/utils"
	corev1 "k8s.io/api/core/v1"
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
		utils.CheckTshLogin()
		utils.CheckPsql()
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		target, kubeObj, exist := utils.DoesInstanceExist(args[0], inCluster)

		if !exist {
			os.Exit(1)
		}

		modeTypes := []string{"read-only", "read-write"}

		modePrompt := promptui.Select{
			Label: "Select DB login mode",
			Items: modeTypes,
		}

		_, mode, err := modePrompt.Run()
		if err != nil {
			return errors.Wrapf(err, "Prompt failed")
		}

		secretObj := &corev1.Secret{}

		switch mode {
		case "read-only":
			err = kubeObj.Client.Get(context.Background(), types.NamespacedName{Name: target.Name + "-rdsiam-readonly.postgres-user-password", Namespace: target.Namespace}, secretObj)
			if err != nil {
				return fmt.Errorf("Error getting secret for instance %s", err)
			}

			clusterName := strings.TrimPrefix(kubeObj.Context, "teleport.aws-us-support.ridecell.io-")
			clusterPrefix := strings.Split(clusterName, ".")[0]
			// Derive RDS instance name using hostname and clusterPrefix
			// We are adding Cluster prefix to RDS instance names, because
			// teleport imported RDS instances with overrided names, so that
			// RDS instances with same name accross the regions can be distinguished.
			// e.g. https://github.com/Ridecell/kubernetes/blob/f994f44ffcc49d6f30f4554c4bcf9a801a05e24b/overlays/aws-eu-prod/summon-uat/summon-uat-rdsinstance.yml#L50-L51
			rdsInstanceName := clusterPrefix + "-" + strings.Split(string(secretObj.Data["host"]), ".")[0]

			pterm.Info.Println("Getting database login credentials")
			dbLoginArgs := []string{"db", "login", "--db-user=" + string(secretObj.Data["username"]), "--db-name=" + string(secretObj.Data["dbname"]), rdsInstanceName}
			err = exec.ExecuteCommand("tsh", dbLoginArgs, false)
			if err != nil {
				return fmt.Errorf("Could not login to database, %s", err)
			}
			pterm.Info.Println("Logging in into database with read-only mode")
			dbConnectCmd := []string{"db", "connect", rdsInstanceName}
			return exec.ExecuteCommand("tsh", dbConnectCmd, true)
		case "read-write":
			pterm.Info.Println("Getting database login credentials")
			// Read application database credentials for login
			err = kubeObj.Client.Get(context.Background(), types.NamespacedName{Name: target.Name + ".postgres-user-password", Namespace: target.Namespace}, secretObj)
			if err != nil {
				return fmt.Errorf("Error getting secret for instance %s", err)
			}

			// Prompt user for confirming read-write mode for Prod/UAT env.
			if target.Env == "prod" || target.Env == "uat" {
				confirmPrompt := promptui.Prompt{
					Label:     "This is " + target.Env + " environment. Make sure you really want to use read-write mode",
					IsConfirm: true,
				}
				goAhead, _ := confirmPrompt.Run()
				if goAhead != "y" {
					os.Exit(0)
				}
			}

			pterm.Warning.Println("Logging in into database with read-write mode")
			// Since RDS is only accesible from kuberntes cluster, executing psql command from a pod in cluster.
			kubectlArgs := []string{"exec", "-it", "-n", "ridectl", "ridectl-helper-0", "--context", kubeObj.Context, "--", "env", "PGPASSWORD=" + string(secretObj.Data["password"]), "psql", "-h", string(secretObj.Data["host"]), "-U", string(secretObj.Data["username"]), string(secretObj.Data["dbname"])}
			return exec.ExecuteCommand("kubectl", kubectlArgs, true)
		}
		return nil
	},
}
