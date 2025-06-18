/*
Copyright 2025 Ridecell, Inc.
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
	rootCmd.AddCommand(proxyCmd)
}

var proxyCmd = &cobra.Command{
	Use:   "proxy <teleport_app_name>",
	Short: "Creates a proxy to Teleport's TCP application to access it localy.",
	Long: "Example:\n" +
		"ridectl proxy <teleport_app_name>               -- e.g. ridectl proxy data-lab-superset-db\n",
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("teleport application name argument is required")
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
			return fmt.Errorf("could not login to teleport app, %s", err)
		}
		pterm.Info.Println("Starting proxy to application")
		appProxyCmd := []string{"proxy", "app", args[0]}
		return exec.ExecuteCommand("tsh", appProxyCmd, true)
	},
}
