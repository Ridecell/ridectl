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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Ridecell/ridectl/pkg/cmd/edit"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/pterm/pterm"
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
			pterm.Warning.Printf("environment variable GOOGLE_SERVICE_ACCOUNT_KEY not defined, skipping image check\n")
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
				pterm.Error.Printf("%s\n", err.Error())
				failedTests = true
			}
		}

		keyPrefixRegex := regexp.MustCompile(strings.ReplaceAll(skipSecretKeysPrefixFlag, ",", "|"))
		for _, locationList := range allSecretLocations {
			if len(locationList) > 1 {
				if skipSecretKeysPrefixFlag != "" {
					if keyPrefixRegex.FindString(locationList[0].KeyName) != "" {
						pterm.Info.Printf("Skipped key '%s' because it starts with a skipped secret key prefix \n", locationList[0].KeyName)
						continue
					}
				}

				failedTests = true

				keysMatch := true
				//var allObjNames []string
				for _, location := range locationList {
					//allObjNames = append(allObjNames, location.ObjName)
					if location.KeyName != locationList[0].KeyName {
						keysMatch = false
					}
				}

				if keysMatch {
					pterm.Info.Printf("Duplicate secret value %s found in %s\n", locationList[0].KeyName, strings.Join(locationList.objNames(), ", "))
				} else {
					pterm.Info.Printf("Duplicate secret value found in %s\n", strings.Join(locationList.formatStrings(), ", "))
				}
			}
		}
		if failedTests {
			pterm.Error.Printf("Tests failed.\n")
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

	if len(manifest) == 0 {
		// return here because we are ignoring old ridecell-operator manifests
		return nil
	}

	// As per our instructions https://github.com/Ridecell/kubernetes-summon#delete-summon-instance, we first remove
	// summon-platform and encryptedsecrets objects from instance.yaml and then delete the yaml later. Below check makes sure
	// lint does not fail even if there in only one namespace object in instance.yaml
	if len(manifest) == 1 && manifest[0].Object.DeepCopyObject().GetObjectKind().GroupVersionKind().Kind == "Namespace" {
		// return here because we are ignoring namespace manifest
		return nil
	}
	// we need to do this because we don't want more than two objects in manifest but we already checked for empty manifest above
	if len(manifest) != 3 {
		return fmt.Errorf("%s: expected three objects in file got %v", filename, len(manifest))
	}

	summonObj, ok := manifest[1].Object.(*summonv1beta2.SummonPlatform)
	if !ok {
		return fmt.Errorf("%s: SummonPlatform is required to be the second object in manifest", filename)
	}
	existingFilename, ok := foundNames[summonObj.Name]
	if ok {
		return fmt.Errorf("Duplicate SummonPlatform names not supported: %s found in %s and %s", summonObj.Name, existingFilename, filename)
	}
	foundNames[summonObj.Name] = filename

	// Make sure that version is set
	if summonObj.Spec.Version == "" {
		return fmt.Errorf("%s: Version is not set.", filename)
	}

	// Make sure that AWS_REGION is set
	if len(summonObj.Spec.Config.Raw) != 0 {
		config := map[string]interface{}{}
		err := json.Unmarshal(summonObj.Spec.Config.Raw, &config)
		if err != nil {
			return fmt.Errorf("%s: unable to deserialize Spec.Config", filename)
		}
		if _, ok := config["AWS_REGION"]; !ok {
			return fmt.Errorf("%s: AWS_REGION is required", filename)
		}
		// Check if DEBUG is set to True; if yes, throw error
		if b := config["DEBUG"]; b != nil && b.(bool) {
			return fmt.Errorf("%s: Please disable the DEBUG flag. More Info: https://www.acunetix.com/vulnerabilities/web/django-debug-mode-enabled/ ", filename)
		}
	} else {
		return fmt.Errorf("%s: AWS_REGION is required", filename)
	}

	// Check that the docker image exists
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

	if manifest[2].Kind != "EncryptedSecret" {
		return fmt.Errorf("%s: EncryptedSecret is required to be the third object in manifest", filename)
	}

	var unencryptedValueFound bool

	for secretKey, secretValue := range manifest[2].Data {
		if !strings.HasPrefix(secretValue, "AQICAH") && !strings.HasPrefix(secretValue, "crypto ") {
			unencryptedValueFound = true
			pterm.Warning.Printf("%s: EncryptedSecret %s missing preamble, may not be encrypted.", filename, secretKey)
		}

		allSecretLocations[secretValue] = append(allSecretLocations[secretValue], secretLocation{ObjName: summonObj.Name, KeyName: secretKey})
	}

	if unencryptedValueFound {
		// Return blank error to not spam terminal, added benefit of spacing out filenames.
		return fmt.Errorf("")
	}

	// Check FERNET_KEYS exists, otherwise return error and exit fn
	_, ok = manifest[2].Data["FERNET_KEYS"]

	if !ok {
		return fmt.Errorf("%s: Key FERNET_KEYS is not present in EncryptedSecret, please refer https://github.com/Ridecell/kubernetes-summon#adding-fernet-keys for help.", filename)
	}

	// Create AWS KMS session to decrypt secret values so we can lint them
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region: aws.String("us-west-1"),
		},
	}))
	kmsService := kms.New(sess)

	err = manifest.Decrypt(kmsService, false)
	if err != nil {
		return fmt.Errorf("Unable to decrypt secret values for %s to lint: %s", summonObj.Name, err)
	}
	// manifest data now has decrypted value; check that FERNET_KEYS is not empty
	if manifest[2].Data["FERNET_KEYS"] == "" {
		return fmt.Errorf("%s's FERNET_KEYS must not be an empty value.\n", summonObj.Name)
	}

	// If customerPortal is enabled, check that GATEWAY_WEB_CLIENT_TOKEN secret is configured.
	// It is required for customerportal to successfully deploy.
	if summonObj.Spec.CustomerPortal.Version != "" && manifest[2].Data["GATEWAY_WEB_CLIENT_TOKEN"] == "" {
		return fmt.Errorf("%s: GATEWAY_WEB_CLIENT_TOKEN must be present if customerPortal is enabled. " +
		  "Please refer to https://github.com/Ridecell/comp-customer-portal#required-kubernetes-config-for-successful-deployment " +
		  "for help.", summonObj.Name)
	}
	// Start checking from the second object in the manifest, ignore the namespace object
	for _, object := range manifest[1:] {
		if object.Meta.GetName() != expectedName {

			return fmt.Errorf("%s: %s name %s did not match expected value %s", filename, object.Kind, object.Meta.GetName(), expectedName)
		}
		if object.Meta.GetNamespace() != clusterEnv && object.Meta.GetNamespace() != fmt.Sprintf("summon-%s", object.Meta.GetName()) {
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
