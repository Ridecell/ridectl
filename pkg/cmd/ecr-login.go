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
	"encoding/base64"
	"strings"

	"github.com/Ridecell/ridectl/pkg/exec"
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

1. Check if existing AWS SSO creds are valid, if not, renew them and obtain credentials for ecr login role.
2. Retrieve AWS ECR login credentials
3. Read existing ~/.docker/config.json file if present, and update/add ECR credentials in it.
4. Save ~/.docker/config.json with updated auth data.

*/

var ecrLoginCmd = &cobra.Command{
	Use:   "ecr-login",
	Short: "AWS ECR registry login",
	Long:  `Logins to AWS ECR container registry`,
	Args: func(_ *cobra.Command, args []string) error {
		return nil
	},
	RunE: func(_ *cobra.Command, fileNames []string) error {

		// Use "ecr-login" SSO role for retrieving ECR credentials
		cfg, err := getAWSConfig("ecr-login", "us-west-2")
		if err != nil {
			return errors.Wrapf(err, "error creating AWS session")
		}

		// Create an Amazon ECR service client
		ecrService := ecr.NewFromConfig(cfg)
		output, err := ecrService.GetAuthorizationToken(context.TODO(), &ecr.GetAuthorizationTokenInput{})
		if err != nil {
			return errors.Wrapf(err, "error creating ECR auth token")
		}

		// Decode Auth Token and extract ECR auth password
		decodedToken, err := base64.StdEncoding.DecodeString(*output.AuthorizationData[0].AuthorizationToken)
		if err != nil {
			return errors.Wrapf(err, "error decoding ECR auth token")
		}
		ecrAuth := strings.TrimPrefix(string(decodedToken), "AWS:")

		dockerArgs := []string{"login", "--username", "AWS", *output.AuthorizationData[0].ProxyEndpoint, "--password", ecrAuth}
		err = exec.ExecuteCommand("docker", dockerArgs, false)
		if err == nil {
			pterm.Success.Println("ECR login successful. NOTE: These credentials are only valid for 12 hours.")
		}
		return err
	},
}
