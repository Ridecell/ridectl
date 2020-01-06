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
	"strings"

	"github.com/spf13/cobra"

	"github.com/Ridecell/ridectl/pkg/kubernetes"
)

func init() {
	rootCmd.AddCommand(lsCmd)
}

var lsCmd = &cobra.Command{
	Use:   "ls [environment]",
	Short: "Lists tenants that ridectl can connect to",
	Long:  "Lists all SummonPlatform instances (or just instances in [environment] that ridectl can connect to/interact with (Note: you may be restricted depending on your permissions)",
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) > 1 {
			return fmt.Errorf("ls takes at most one optional argument: [environment]")
		}
		return nil
	},
	RunE: func(_ *cobra.Command, args []string) error {
		namespaces := []string{"summon-qa", "summon-dev", "summon-uat", "summon-prod"}
		env := strings.ToLower(args[0])
		// If user listed an environment, only get tenants in that environment
		if strings.HasSuffix(env, "qa") {
			namespaces = []string{"summon-qa"}
		}
		if strings.HasSuffix(env, "dev") {
			namespaces = []string{"summon-dev"}
		}
		if strings.HasSuffix(env, "uat") {
			namespaces = []string{"summon-uat"}
		}
		if strings.HasSuffix(env, "prod") {
			namespaces = []string{"summon-prod"}
		}

		for _, namespace := range namespaces {
			instances, err := kubernetes.ListSummonPlatforms(kubeconfigFlag, namespace)
			if err != nil {
				continue
			}

			fmt.Printf("\n%s\n=========================\n", strings.ToUpper(namespace))
			for _, instance := range instances.Items {
				fmt.Printf("%s\n", instance.Name)
			}
		}
		return nil
	},
}
