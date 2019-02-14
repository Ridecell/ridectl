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
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Ridecell/ridectl/pkg/cmd/edit"
	"github.com/spf13/cobra"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	rootCmd.AddCommand(lintCmd)
}

var lintCmd = &cobra.Command{
	Use:   "lint [flags]",
	Short: "Lints SummonPlatform manifest files",
	Long:  `Checks Summon instance manifest files for invalid values and names`,
	Args:  func(_ *cobra.Command, args []string) error { return nil },
	RunE: func(_ *cobra.Command, args []string) error {

		fileNames, err := walkDir()
		if err != nil {
			return err
		}

		var failedTests bool
		for _, filename := range fileNames {
			err = parseFile(filename)
			if err != nil {
				fmt.Printf("%s\n", err.Error())
				failedTests = true
			}
		}
		if failedTests {
			fmt.Printf("Tests failed.\n")
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

func parseFile(filename string) error {
	path, file := filepath.Split(filename)

	clusterEnv := filepath.Base(path)
	clusterName := strings.Split(file, ".")[0]

	// Our expected name should be filename-foldername
	expectedName := fmt.Sprintf("%s-%s", clusterName, clusterEnv)

	// Check our filename against expected values
	match := regexp.MustCompile(`^[a-z0-9]+.yml`).Match([]byte(file))
	if !match {
		// Other checks not reliable if this fails, continue
		return fmt.Errorf("%s: invalid file name, must match ^[a-z0-9]+.yml$", filename)
	}

	match = regexp.MustCompile(`^[a-z0-9]+-[a-z]+$`).Match([]byte(expectedName))
	if !match {
		return fmt.Errorf("%s: got invalid name-env %s", filename, expectedName)
	}

	manifest, err := getManifest(filename)
	if err != nil {
		return err
	}
	// Only do parsing checks on shared.yml
	if file == "shared.yml" {
		return nil
	}

	for count, object := range manifest {
		switch count {
		case 0:
			_, ok := object.Object.(*summonv1beta1.SummonPlatform)
			if !ok {
				return fmt.Errorf("%s: SummonPlatform is required to be the first object in manifest", filename)
			}
			object.Meta, ok = object.Object.(metav1.Object)
			if !ok {
				return fmt.Errorf("%s: failed to get metdata for SummonPlatform object", filename)
			}
		case 1:
			if object.Kind != "EncryptedSecret" {
				return fmt.Errorf("%s: EncryptedSecret is required to be the second object in manifest", filename)
			}
		case 2:
			return fmt.Errorf("%s: more than two objects found, exiting", filename)
		}
		if object.Meta.GetName() != expectedName {
			return fmt.Errorf("%s: %s name %s did not match expected value %s", filename, object.Kind, object.Meta.GetName(), expectedName)
		}
		if object.Meta.GetNamespace() != clusterEnv {
			return fmt.Errorf("%s: %s namespace %s did not match expected value %s", filename, object.Kind, object.Meta.GetNamespace(), clusterEnv)
		}
	}
	return nil
}

func walkDir() ([]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	var fileNames []string
	err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
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
