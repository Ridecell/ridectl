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

	dbv1beta2 "github.com/Ridecell/ridecell-controllers/apis/db/v1beta2"
	secretsv1beta2 "github.com/Ridecell/ridecell-controllers/apis/secrets/v1beta2"
	hackapis "github.com/Ridecell/ridectl/pkg/apis"
	summonv1beta2 "github.com/Ridecell/summon-operator/apis/app/v1beta2"
)

var (
	kubeconfigFlag string
	versionFlag    bool
	version        string
	inCluster      bool
)
var rootCmd = &cobra.Command{
	Use:           "ridectl",
	Short:         "Ridectl controls Summon instances in Kubernetes",
	SilenceErrors: true,
	RunE: func(_ *cobra.Command, args []string) error {
		if versionFlag {
			pterm.Success.Printf("ridectl version %s\n", version)
		} else if len(args) == 0 {

			return fmt.Errorf("No command specified.")
		}
		return nil
	},
}

type versionInfo struct {
	Name    string `json:"name"`
	TagName string `json:"tag_name"`
}

func init() {
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	rootCmd.PersistentFlags().StringVar(&kubeconfigFlag, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	rootCmd.Flags().BoolVar(&versionFlag, "version", false, "--version")
	rootCmd.PersistentFlags().BoolVar(&inCluster, "incluster", false, "(optional) use in cluster kube config")
	// check version and update if not latest
	if !isLatestVersion() {
		updatePrompt := promptui.Prompt{
			Label:     "Do you want to update to latest version",
			IsConfirm: true,
		}
		shouldUpdate, _ := updatePrompt.Run()
		if shouldUpdate == "y" {
			selfUpdate()
		}
	}
	// Register all types from summon-operator and ridecell-controllers secrets
	_ = summonv1beta2.AddToScheme(scheme.Scheme)
	_ = secretsv1beta2.AddToScheme(scheme.Scheme)
	_ = hackapis.AddToScheme(scheme.Scheme)
	_ = dbv1beta2.AddToScheme(scheme.Scheme)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		pterm.Error.Println(err)
		pterm.Error.Println("For FAQs and Troubleshooting: https://ridecell.quip.com/CILaAnAUnkla/Ridectl-FAQs")
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

	var data versionInfo
	var retry bool
	err = json.Unmarshal(body, &data)
	if err != nil {
		pterm.Error.Printf("Failed to parse version info: %s\n", err)
		retry = true
	}
	// added retry to handle github api not returning proper json
	if retry {
		return isLatestVersion()
	}

	if version != data.TagName {
		pterm.Warning.Printf("You are running older version of ridectl %s\n", version)
		return false
	}

	return true
}

func selfUpdate() {
	var url string
	p := pterm.DefaultProgressbar.WithTotal(3)
	p.ShowElapsedTime = false

	switch runtime.GOOS {

	case "darwin":
		url = "https://github.com/Ridecell/ridectl/releases/latest/download/ridectl_macos.zip"

	case "linux":
		url = "https://github.com/Ridecell/ridectl/releases/latest/download/ridectl_linux.zip"

	}

	_, _ = p.Start()
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
		pterm.Error.Printf("Failed to create buffer for zip file: %s\n", err)
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
		pterm.Error.Printf("Failed to update binary: %s\n", err)
		pterm.Info.Printf("If it's a permission related issue, then please re-run ridectl with sudo privileges to update")
		os.Exit(1)
	}
	p.Increment()
	_, _ = p.Stop()

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
