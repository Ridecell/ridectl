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
	Long: "Lists all SummonPlatform instances or just instances in [environment] that ridectl" +
		"can connect to/interact with. Note: you may be restricted depending on your permissions.\n" +
		"Examples:\n\tridectl ls\n\tridectl ls dev\n\tridectl ls darwin-qa",
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) > 1 {
			return fmt.Errorf("ls takes at most one optional argument: [environment]")
		}
		return nil
	},
	RunE: func(_ *cobra.Command, args []string) error {
		namespaces := []string{"summon-qa", "summon-dev", "summon-uat", "summon-prod"}
		var nameregex string
		if len(args) == 1 {
			search := strings.ToLower(args[0])
			nameregex = search
			// If user listed an environment, only get tenants in that environment
			if strings.HasSuffix(search, "qa") {
				namespaces = []string{"summon-qa"}
			} else if strings.HasSuffix(search, "dev") {
				namespaces = []string{"summon-dev"}
			} else if strings.HasSuffix(search, "uat") {
				namespaces = []string{"summon-uat"}
			} else if strings.HasSuffix(search, "prod") {
				namespaces = []string{"summon-prod"}
			} else {
				return fmt.Errorf("%s not found or recognized.\n", search)
			}

			// If arg was just "qa", 'dev", "uat", "prod" or "summon-<env>",the then we actually
			// want to set nameregex to empty so ListSummonPlatform will traverse the proper code path.
			for _, env := range []string{"summon-qa", "summon-dev", "summon-uat", "summon-prod", "qa", "dev", "uat", "prod"} {
				if nameregex == env {
					nameregex = ""
				}
			}
		}

		for _, namespace := range namespaces {
			instances, err := kubernetes.ListSummonPlatforms(kubeconfigFlag, nameregex, namespace)
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
