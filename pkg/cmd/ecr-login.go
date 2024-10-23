/*
Copyright 2024 Ridecell, Inc.

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
	"encoding/json"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/pterm/pterm"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(ecrLoginCmd)
}

/*

An explanation of the ECR login process:

1. Check if existing AWS SSO creds are valid, if not, renew them and obtain credentials for docker login role.
2. Retrieve AWS ECR login credentials
3. Read existing ~/.docker/config.json file if present, and update/add ECR credentials in it.
4. Save ~/.docker/config.json with updated auth data.

*/

var ecrLoginCmd = &cobra.Command{
	Use:   "ecr-login",
	Short: "AWS ECR registry login",
	Long:  `Logins to AWS ECR docker registry`,
	Args: func(_ *cobra.Command, args []string) error {
		return nil
	},
	RunE: func(_ *cobra.Command, fileNames []string) error {

		cfg, err := createAWSConfig("devops-team", "us-west-2")
		if err != nil {
			return errors.Wrapf(err, "error creating AWS session")
		}

		// Create an Amazon KMS service client
		ecrService := ecr.NewFromConfig(cfg)
		output, err := ecrService.GetAuthorizationToken(context.TODO(), &ecr.GetAuthorizationTokenInput{})
		if err != nil {
			return errors.Wrapf(err, "error creating ECR auth token")
		}

		// Create docker creds using ECR Auth token output
		ecrAuth := map[string]string{
			"auth": *output.AuthorizationData[0].AuthorizationToken,
		}

		// Load existing ~/.docker/config.json file if exists, create if not present.
		userHomeDir, err := os.UserHomeDir()
		if err != nil {
			pterm.Error.Printf("error getting user home directory: %v", err)
			os.Exit(1)
		}
		dockerDir := userHomeDir + "/.docker"
		createDirIfNotPresent(dockerDir)

		// Load existing ~/.docker/config.json file if exists
		dockerConfig := map[string]interface{}{}
		configData, _ := os.ReadFile(dockerDir + "/config.json")
		if configData != nil {
			_ = json.Unmarshal(configData, &dockerConfig)
		}

		// Add/Update docker creds
		if _, ok := dockerConfig["auths"]; ok {
			dockerConfig["auths"].(map[string]interface{})[*output.AuthorizationData[0].ProxyEndpoint] = ecrAuth
		} else {
			dockerConfig["auths"] = map[string]interface{}{
				*output.AuthorizationData[0].ProxyEndpoint: ecrAuth,
			}
		}

		byteValue, err := json.MarshalIndent(dockerConfig, "", "  ")
		if err != nil {
			return err
		}
		err = os.WriteFile(dockerDir+"/config.json", byteValue, 0600)
		if err == nil {
			pterm.Success.Println("ECR login successful. NOTE: These credentials are only valid for 12 hours.")
		}
		return err
	},
}
