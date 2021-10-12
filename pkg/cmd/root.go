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
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes/scheme"

	secretsv1beta2 "github.com/Ridecell/ridecell-controllers/apis/secrets/v1beta2"
	hackapis "github.com/Ridecell/ridectl/pkg/apis"
	summonv1beta2 "github.com/Ridecell/summon-operator/apis/app/v1beta2"
)

var kubeconfigFlag string
var versionFlag bool
var version string

var rootCmd = &cobra.Command{
	Use:   "ridectl",
	Short: "Ridectl controls Summon instances in Kubernetes",
	RunE: func(_ *cobra.Command, args []string) error {
		if versionFlag {
			fmt.Printf("ridectl version %s\n", version)
		}
		return nil
	},
}

func init() {
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	rootCmd.PersistentFlags().StringVar(&kubeconfigFlag, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	rootCmd.Flags().BoolVar(&versionFlag, "version", true, "--version")
	// check version
	if !isLatestVersion() {
		fmt.Println("ridectl version is not latest")
	}
	// Register all types from summon-operator and ridecell-controllers secrets
	_ = summonv1beta2.AddToScheme(scheme.Scheme)
	_ = secretsv1beta2.AddToScheme(scheme.Scheme)
	_ = hackapis.AddToScheme(scheme.Scheme)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func isLatestVersion() bool {
	resp, err := http.Get("https://api.github.com/repos/Ridecell/ridectl/releases/latest")
	if err != nil {
		log.Fatalln(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var data1 interface{}
	err = json.Unmarshal(body, &data1)
	if err != nil {
		panic(err.Error())
	}

	return version == data1.(map[string]interface{})["tag_name"].(string)
}
