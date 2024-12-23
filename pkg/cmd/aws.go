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
	"regexp"
	"strings"

	"github.com/Ridecell/ridectl/pkg/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
)

func createAWSConfig(roleName, region string) (aws.Config, error) {
	var cfg aws.Config

	// If no-aws-sso flag is provided, do not use AWS SSO creds, instead load default configuration.
	if noAWSSSO {
		// Load the Shared AWS Configuration (~/.aws/config)
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
		if err != nil {
			err = errors.Wrapf(err, "error creating AWS session")
		}
		return cfg, err
	}

	updateAWSAccountInfo := false
	startUrl, accountId := utils.LoadAWSAccountInfo(ridectlConfigFile)
	if startUrl == "" || accountId == "" {
		updateAWSAccountInfo = true

		prompt := promptui.Prompt{
			Label:    "Enter AWS SSO Start url",
			Validate: validateStartUrl,
		}
		var err error
		startUrl, err = prompt.Run()
		if err != nil {
			return cfg, errors.Wrapf(err, "Prompt failed")
		}

		prompt = promptui.Prompt{
			Label:    "Enter AWS Account ID",
			Validate: validateAccountId,
		}
		accountId, err = prompt.Run()
		if err != nil {
			return cfg, errors.Wrapf(err, "Prompt failed")
		}
	}

	// Retrieve AWS SSO credentials for roleName
	credentialsPath := utils.RetriveAWSSSOCredsPath(ridectlHomeDir, startUrl, accountId, roleName)

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedCredentialsFiles([]string{credentialsPath}),
		config.WithSharedConfigProfile(roleName),
		config.WithDefaultRegion(region))
	if err != nil {
		err = errors.Wrapf(err, "error retrieving AWS SSO credentials")
	}
	if updateAWSAccountInfo {
		// Create/Update AWS Account info
		err = utils.UpdateAWSAccountInfo(ridectlConfigFile, startUrl, accountId)
	}
	return cfg, err
}

func validateStartUrl(input string) error {
	if strings.Contains(input, " ") {
		return errors.New("Remove white-spaces from input [" + input + "]")
	}

	pattern := `^https://[a-zA-Z0-9-]+\.awsapps\.com/start$`
	matched, err := regexp.MatchString(pattern, input)
	if err != nil {
		return err
	}
	if !matched {
		return errors.New("start url is invalid")
	}
	return nil
}
func validateAccountId(input string) error {
	pattern := `^[0-9]{12}$`
	matched, err := regexp.MatchString(pattern, input)
	if err != nil {
		return err
	}
	if !matched {
		return errors.New("aws account id is invalid")
	}
	return nil
}
