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
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Ridecell/ridectl/pkg/cmd/edit"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/spf13/cobra"

	summonv1beta2 "github.com/Ridecell/summon-operator/apis/app/v1beta2"
)

var skipSecretKeysPrefixFlag string

func init() {
	rootCmd.AddCommand(lintCmd)
	lintCmd.Flags().StringVarP(&skipSecretKeysPrefixFlag, "skipSecretKeysPrefix", "s", "", "enter prefixes of keys which need to be skipped (as comma separated values)")
}

type secretLocation struct {
	ObjName string
	KeyName string
}

type secretLocations []secretLocation

func (sl secretLocations) objNames() []string {
	var allObjNames []string
	for _, location := range sl {
		allObjNames = append(allObjNames, location.ObjName)
	}
	return allObjNames
}

func (sl secretLocations) formatStrings() []string {
	var allFormattedStrings []string
	for _, location := range sl {
		allFormattedStrings = append(allFormattedStrings, fmt.Sprintf("%s: %s", location.ObjName, location.KeyName))
	}
	return allFormattedStrings
}

var foundNames map[string]string
var allSecretLocations map[string]secretLocations

var lintCmd = &cobra.Command{
	Use:   "lint [flags] <path>...",
	Short: "Lints SummonPlatform manifest files",
	Long:  `Checks Summon instance manifest files for invalid values and names`,
	Args:  func(_ *cobra.Command, args []string) error { return nil },
	RunE: func(_ *cobra.Command, args []string) error {
		// Fetch docker image names
		googleKey := os.Getenv("GOOGLE_SERVICE_ACCOUNT_KEY")
		if len(googleKey) == 0 {
			fmt.Printf("environment variable GOOGLE_SERVICE_ACCOUNT_KEY not defined, skipping image check\n")
		}

		var imageTags []string
		var err error
		if len(googleKey) > 0 {
			transport := registry.WrapTransport(http.DefaultTransport, "https://us.gcr.io", "_json_key", googleKey)
			hub := &registry.Registry{
				URL: "https://us.gcr.io",
				Client: &http.Client{
					Transport: transport,
				},
				Logf: registry.Quiet,
			}

			imageTags, err = hub.Tags("ridecell-1/summon")
			if err != nil {
				return err
			}
		}

		foundNames = make(map[string]string)
		allSecretLocations = make(map[string]secretLocations)
		var fileNames []string
		if len(args) > 0 {
			fileNames, err = parseArgs(args)
			if err != nil {
				return err
			}
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			fileNames, err = walkDir(cwd)
			if err != nil {
				return err
			}
		}

		var failedTests bool
		for _, filename := range fileNames {
			err = lintFile(filename, imageTags)
			if err != nil {
				fmt.Printf("%s\n", err.Error())
				failedTests = true
			}
		}

		keyPrefixRegex := regexp.MustCompile(strings.ReplaceAll(skipSecretKeysPrefixFlag, ",", "|"))
		for _, locationList := range allSecretLocations {
			if len(locationList) > 1 {
				if skipSecretKeysPrefixFlag != "" {
					if keyPrefixRegex.FindString(locationList[0].KeyName) != "" {
						fmt.Printf("Skipped key '%s' because it starts with a skipped secret key prefix \n", locationList[0].KeyName)
						continue
					}
				}

				failedTests = true

				keysMatch := true
				var allObjNames []string
				for _, location := range locationList {
					allObjNames = append(allObjNames, location.ObjName)
					if location.KeyName != locationList[0].KeyName {
						keysMatch = false
					}
				}

				if keysMatch {
					fmt.Printf("Duplicate secret value %s found in %s\n", locationList[0].KeyName, strings.Join(locationList.objNames(), ", "))
				} else {
					fmt.Printf("Duplicate secret value found in %s\n", strings.Join(locationList.formatStrings(), ", "))
				}
			}
		}
		if failedTests {
			fmt.Printf("Tests failed.\n")
			// Exit here and don't return error so Cobra doesn't display extra text
			os.Exit(1)
		}
		return nil
	},
}

func getManifest(filename string) (edit.Manifest, error) {
	// Read the file in.
	inFile, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", filename, err)
	}
	defer inFile.Close()
	// Parse the input file to objects.
	inManifest, err := edit.NewManifest(inFile)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", filename, err)
	}
	return inManifest, nil
}

func lintFile(filename string, imageTags []string) error {
	path, file := filepath.Split(filename)

	clusterEnv := filepath.Base(path)
	clusterName := strings.Split(file, ".")[0]

	// Our expected name should be filename-foldername
	if strings.Contains(clusterEnv, "-") {
		clusterEnv = strings.Split(clusterEnv, "-")[1]
	}
	expectedName := fmt.Sprintf("%s-%s", clusterName, clusterEnv)

	// Check our filename against expected values
	match := regexp.MustCompile(`^[a-z0-9]+.yml`).Match([]byte(file))
	if !match {
		// Other checks not reliable if this fails, continue
		return fmt.Errorf("%s: invalid file name, must match ^[a-z0-9]+.yml$", filename)
	}

	// Make sure the directory name is valid
	match = regexp.MustCompile(`^[a-z]+-[a-z]+|[a-z]+$`).Match([]byte(clusterEnv))
	if !match {
		return fmt.Errorf("%s: got invalid directory name %s", filename, clusterEnv)
	}

	manifest, err := getManifest(filename)
	if err != nil {
		return err
	}
	// Only do parsing checks on shared.yml
	if file == "shared.yml" {
		return nil
	}

	if len(manifest) != 2 {
		return fmt.Errorf("%s: expected two objects in file got %v", filename, len(manifest))
	}

	summonObj, ok := manifest[0].Object.(*summonv1beta2.SummonPlatform)
	if !ok {
		return fmt.Errorf("%s: SummonPlatform is required to be the first object in manifest", filename)
	}
	existingFilename, ok := foundNames[summonObj.Name]
	if ok {
		return fmt.Errorf("Duplicate SummonPlatform names not supported: %s found in %s and %s", summonObj.Name, existingFilename, filename)
	}
	foundNames[summonObj.Name] = filename

	// Make sure that either autodeploy or version is set
	if summonObj.Spec.AutoDeploy == "" && summonObj.Spec.Version == "" {
		return fmt.Errorf("%s: Neither Autodeploy or Version are set.", filename)
	}

	// Make sure that autodeploy and version are not both set
	if summonObj.Spec.AutoDeploy != "" && summonObj.Spec.Version != "" {
		return fmt.Errorf("%s: Autodeploy and Version both set, only one should be set at a time.", filename)
	}

	// Check that the docker image exists
	if summonObj.Spec.AutoDeploy == "" {
		if imageTags != nil {
			var foundImage bool
			for _, imageTag := range imageTags {
				if summonObj.Spec.Version == imageTag {
					foundImage = true
					break
				}
			}

			if !foundImage {
				return fmt.Errorf(`%s: version "%s" does not exist`, filename, summonObj.Spec.Version)
			}
		}
	}

	if manifest[1].Kind != "EncryptedSecret" {
		return fmt.Errorf("%s: EncryptedSecret is required to be the second object in manifest", filename)
	}

	var fernetKeyFound bool
	var unencryptedValueFound bool
	for secretKey, secretValue := range manifest[1].Data {
		if !strings.HasPrefix(secretValue, "AQICAH") && !strings.HasPrefix(secretValue, "crypto ") {
			unencryptedValueFound = true
			fmt.Printf("%s: EncryptedSecret %s missing preamble, may not be encrypted.", filename, secretKey)
		}

		// Check if FERNET_KEYS key is present or not, as its required in all summon yaml.
		if secretKey == "FERNET_KEYS" {
			fernetKeyFound = true
		}

		allSecretLocations[secretValue] = append(allSecretLocations[secretValue], secretLocation{ObjName: summonObj.Name, KeyName: secretKey})
	}

	if unencryptedValueFound {
		// Return blank error to not spam terminal, added benefit of spacing out filenames.
		return fmt.Errorf("")
	}

	if !fernetKeyFound {
		return fmt.Errorf("%s: Key FERNET_KEYS is not present in EncryptedSecret, please refer https://github.com/Ridecell/kubernetes-summon#adding-fernet-keys for help.", filename)
	}

	for _, object := range manifest {
		if object.Meta.GetName() != expectedName {
			return fmt.Errorf("%s: %s name %s did not match expected value %s", filename, object.Kind, object.Meta.GetName(), expectedName)
		}
		if object.Meta.GetNamespace() != clusterEnv && object.Meta.GetNamespace() != fmt.Sprintf("summon-%s", clusterEnv) {
			return fmt.Errorf("%s: %s namespace %s did not match expected value %s", filename, object.Kind, object.Meta.GetNamespace(), clusterEnv)
		}
	}
	return nil
}

func parseArgs(args []string) ([]string, error) {
	var output []string
	for _, arg := range args {
		// If our input is a directory walk it and append to output
		fileInfo, err := os.Stat(arg)
		if err != nil {
			return nil, err
		}
		if fileInfo.IsDir() {
			files, err := walkDir(arg)
			if err != nil {
				return nil, err
			}
			output = append(output, files...)
		}

		_, filename := filepath.Split(arg)
		// Only care about .yml files and skips hidden files
		if strings.HasSuffix(filename, ".yml") && !strings.HasPrefix(filename, ".") {
			output = append(output, arg)
		}
	}
	return output, nil
}

func walkDir(startDir string) ([]string, error) {
	var fileNames []string
	err := filepath.Walk(startDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip hidden folders
		if strings.HasPrefix(info.Name(), ".") && info.IsDir() {
			return filepath.SkipDir
		}
		// Only care about .yml files and skips hidden files
		if strings.HasSuffix(info.Name(), ".yml") && !strings.HasPrefix(info.Name(), ".") {
			fileNames = append(fileNames, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return fileNames, nil
}
