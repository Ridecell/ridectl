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
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/inconshreveable/go-update"
	"github.com/manifoldco/promptui"
	"github.com/mitchellh/go-homedir"
	"github.com/pterm/pterm"
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
	// check version and update if not latest
	if !isLatestVersion() {
		updatePrompt := promptui.Prompt{
			Label:     "Do you want to update to latest version",
			IsConfirm: true,
		}
		shouldUpdate, _ := updatePrompt.Run()
		fmt.Println(shouldUpdate)
		if shouldUpdate == "y" {
			selfUpdate()
		}
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

	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		panic(err.Error())
	}

	if version != data.(map[string]interface{})["tag_name"].(string) {
		fmt.Printf("You are running older version of ridectl %s\n", version)
		return false
	}

	return true
}

func selfUpdate() {
	var url string
	p := pterm.DefaultProgressbar.WithTotal(3)

	switch runtime.GOOS {

	case "darwin":
		url = "https://github.com/Ridecell/ridectl/releases/latest/download/ridectl_macos.zip"

	case "linux":
		url = "https://github.com/Ridecell/ridectl/releases/latest/download/ridectl_linux.zip"

	}

	p.Start()
	p.Title = "Downloading"
	res, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()
	p.Increment()

	p.Title = "Extracting"
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("Failed to create buffer for zip file: %s\n", err)
	}

	r, err := getBinary(buf)
	if err != nil {
		log.Fatalln(err)
	}
	p.Increment()

	executable, err := os.Executable()
	if err != nil {
		panic(err)
	}

	p.Title = "Installing"
	cmdPath := filepath.Join(executable)
	err = update.Apply(r, update.Options{TargetPath: cmdPath})
	if err != nil {
		fmt.Printf("Failed to update binary: %s\n", err)
	}
	p.Increment()
	p.Stop()

}

func getBinary(src []byte) (io.Reader, error) {
	r := bytes.NewReader(src)
	z, err := zip.NewReader(r, r.Size())
	if err != nil {
		return nil, fmt.Errorf("failed to uncompress zip file: %s", err)
	}
	for _, file := range z.File {
		return file.Open()
	}
	return nil, fmt.Errorf("failed to find binary in zip file")
}
