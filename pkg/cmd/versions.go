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
	"bytes"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/heroku/docker-registry-client/registry"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionsCmd)
}

type parsedTag struct {
	tag, sha, branch string
	build            int
}

type byBuild []parsedTag

func (a byBuild) Len() int           { return len(a) }
func (a byBuild) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byBuild) Less(i, j int) bool { return a[i].build > a[j].build }

var versionsCmd = &cobra.Command{
	Use:   "versions [flags] [branch]",
	Short: "Display available Summon Platform image versions",
	Long:  `Display available Summon Platform image versions for master and release branches, all all recent images for a specific branch`,
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) > 1 {
			return fmt.Errorf("Too many arguments")
		}
		return nil
	},
	RunE: func(_ *cobra.Command, args []string) error {
		// Get a new GCloud access token.
		cmd := exec.Command("gcloud", "config", "config-helper", "--format=value(credential.access_token)")
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			return err
		}

		// Connect to the image registry.
		password := strings.TrimSpace(out.String())
		transport := registry.WrapTransport(http.DefaultTransport, "https://us.gcr.io", "_dcgcloud_token", password)
		hub := &registry.Registry{
			URL: "https://us.gcr.io",
			Client: &http.Client{
				Transport: transport,
			},
			Logf: registry.Quiet,
		}

		// Get all tags for the summon image.
		tags, err := hub.Tags("ridecell-1/summon")
		if err != nil {
			return err
		}

		// Do some quick parsing.
		var parsedTags []parsedTag
		for _, tag := range tags {
			parts := regexp.MustCompile(`^(\d+)-([0-9a-f]+)-(.*)$`).FindStringSubmatch(tag)
			if parts == nil {
				// Not sure what that is.
				continue
			}
			build, err := strconv.Atoi(parts[1])
			if err != nil {
				panic(err)
			}
			parsedTags = append(parsedTags, parsedTag{tag: tag, build: build, sha: parts[2], branch: parts[3]})
		}

		// Check which mode we are in.
		if len(args) == 0 {
			// Show the latest build on important branches (master, ^release)
			byBranch := map[string]parsedTag{}
			for _, parsed := range parsedTags {
				existing, ok := byBranch[parsed.branch]
				if !ok || parsed.build > existing.build {
					byBranch[parsed.branch] = parsed
				}
			}
			branchRegexp := regexp.MustCompile(`^(master$|release)`)
			branches := make([]string, 0, len(byBranch))
			for b := range byBranch {
				if branchRegexp.MatchString(b) {
					branches = append(branches, b)
				}
			}
			sort.Strings(branches)
			for _, b := range branches {
				parsed := byBranch[b]
				fmt.Printf("%s: %s\n", parsed.branch, parsed.tag)
			}

		} else {
			// Show the past 10 builds on a branch matching this substring.
			branchRegexp, err := regexp.Compile(args[0])
			if err != nil {
				return err
			}
			matchingTags := byBuild{}
			for _, parsed := range parsedTags {
				if branchRegexp.MatchString(parsed.branch) {
					matchingTags = append(matchingTags, parsed)
				}
			}
			sort.Sort(matchingTags)
			for i, parsed := range matchingTags {
				fmt.Println(parsed.tag)
				if i > 10 {
					break
				}
			}
		}

		return nil
	},
}
