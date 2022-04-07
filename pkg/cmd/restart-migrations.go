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
	"github.com/spf13/cobra"
	"github.com/pterm/pterm"
)

func init() {
	rootCmd.AddCommand(restartMigrationsCmd)
}

var restartMigrationsCmd = &cobra.Command{
	Use:   "restart-migrations [flags] <cluster_name> ",
	Short: "Restart migrations for target summon instance.",
	Long: "Restart migrations for target summon instance.\n" +
		"restart-migrations <instance> e.g ridectl restart-migrations summontest-dev",
	Args: func(_ *cobra.Command, args []string) error {
		return nil
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		pterm.Info.Println("Ridectl restart-migrations command is deprecated, and will be removed in later releases.\nPlease use 'ridectl restart' command instead.")
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}
