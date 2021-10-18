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

package utils

import (
	"flag"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pterm/pterm"

	"k8s.io/client-go/util/homedir"
)

func GetKubeconfig() *string {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	return kubeconfig
}

func CheckBinary(binary string) bool {
	_, err := exec.LookPath(binary)
	return err == nil
}

func CheckVPN() {
	resp, err := http.Head("https://ridectl.s3.us-west-2.amazonaws.com/machinload01.png")
	if err != nil {
		pterm.Error.Println("\n", err)
	}
	if resp.StatusCode != 200 {
		pterm.Error.Println("VPN is not connected")
		os.Exit(1)
	}
}
