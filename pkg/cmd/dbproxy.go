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
	"fmt"

	"github.com/Ridecell/ridectl/pkg/exec"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	utils "github.com/Ridecell/ridectl/pkg/utils"
)

func init() {
	rootCmd.AddCommand(dbProxyCmd)
}

var dbProxyCmd = &cobra.Command{
	Use:   "dbproxy <app_database_name>",
	Short: "Creates a proxy to IBM application's database instance to access it localy.",
	Long: "Example:\n" +
		"ridectl dbproxy <app_database_name>               -- e.g. ridectl dbproxy data-lab-superset-db\n",
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("application database name argument is required")
		}
		if len(args) > 1 {
			return fmt.Errorf("too many arguments")
		}
		return nil
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		utils.CheckTshLogin()
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		pterm.Info.Println("Logging into app")
		appLoginArgs := []string{"apps", "login", args[0]}
		err := exec.ExecuteCommand("tsh", appLoginArgs, false)
		if err != nil {
			return fmt.Errorf("could not login to database app, %s", err)
		}
		pterm.Info.Println("Starting proxy to database")
		appProxyCmd := []string{"proxy", "app", args[0]}
		return exec.ExecuteCommand("tsh", appProxyCmd, true)
	},
}
