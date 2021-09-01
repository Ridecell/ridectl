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
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"

	kubernetes "github.com/Ridecell/ridectl/pkg/kubernetes"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newCmd)
}

var newCmd = &cobra.Command{
	Use:   "new <instance_name>",
	Short: "Create a new summon-platform.yml for given instance",
	Long: "Create a new summon-platform.yml for given instance.\n" +
		"For summon instances: new <instance_name>	-- e.g. ridectl new summontest-dev\n",
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("cluster name argument is required")
		}
		if len(args) > 1 {
			return fmt.Errorf("too many arguments")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		target, err := kubernetes.ParseSubject(args[0])
		if err != nil {
			return errors.Wrapf(err, "not a valid target %s", args[0])
		}
		fmt.Printf("this is target: \n%+v\n", target)

		newInstance, err := TempFS.ReadFile("templates/new_instance.yml.tpl")
		if err != nil {
			return errors.Wrap(err, "error reading new_instance.yml.tpl")
		}

		template, err := template.New("new_instance.yml.tpl").Parse(string(newInstance))
		if err != nil {
			return errors.Wrap(err, "error parsing new_instance.yml.tpl")
		}

		// Prompt user for a slack channel to alert to
		slackChannelPrompt := promptui.Prompt{
			Label: "Enter a slack channel name (#channel-name, blank to skip)",
			Validate: func(input string) error {
				if !strings.HasPrefix(input, "#") && input != "" {
					return errors.New(`Channel name must have prefix "#"`)
				}
				return nil
			},
		}

		slackChannelNames, err := slackChannelPrompt.Run()
		if err != nil {
			return err
		}

		slackChannels := strings.Split(slackChannelNames, ",")
		buffer := &bytes.Buffer{}
		err = template.Execute(buffer, struct {
			Name          string
			Namespace     string
			SlackChannels []string
		}{
			Name:          args[0],
			Namespace:     target.Namespace,
			SlackChannels: slackChannels,
		})
		if err != nil {
			return errors.Wrap(err, "error executing template")
		}

		match := regexp.MustCompile(`^([a-z0-9]+)-([a-z]+)$`).FindStringSubmatch(args[0])
		newInstnaceFile, err := os.Create(match[1] + ".yml")
		if err != nil {
			return errors.Wrap(err, "error creating file")
		}

		// Write the contents of the buffer to the file instance-name.yml
		_, err = newInstnaceFile.Write(buffer.Bytes())
		if err != nil {
			return errors.Wrap(err, "error writing to file")
		}

		return nil
	},
}
