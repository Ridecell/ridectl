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

	"github.com/Ridecell/ridecell-operator/pkg/webui/kubernetes"
	//summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	//"github.com/Ridecell/ridectl/pkg/kubernetes"
	//dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	//corev1 "k8s.io/api/core/v1"
)

func init() {
	rootCmd.AddCommand(lsCmd)
}

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "Lists tenants that ridectl can connect to",
	Long:  "Lists all SummonPlatform instances that ridectl can connect to/interact with (Note: you may be restricted depending on your permissions)",
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("ls does not take any arguments")
		}
		return nil
	},
	RunE: func(_ *cobra.Command, args []string) error {
		namespaces := []string{"summon-qa", "summon-dev", "summon-uat", "summon-prod"}
		for _, namespace := range namespaces {
			instances, err := kubernetes.ListSummonPlatform(namespace)
			if err != nil {
				continue
			}
			for _, instance := range instances.Items {
				fmt.Printf("%s", instance.Name)
			}
		}
		return nil
	},
}
