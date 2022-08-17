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
	"path/filepath"
	"strings"

	"github.com/Ridecell/ridectl/pkg/exec"
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

func CheckKubectl() {
	if !exec.CheckBinary("kubectl") {
		pterm.Error.Printf("kubectl is not installed. Follow the instructions here: https://kubernetes.io/docs/tasks/tools/#kubectl to install it\n")
		os.Exit(1)
	}
}

func CheckPsql() {
	if !exec.CheckBinary("psql") {
		pterm.Error.Printf("psql is not installed. Follow the instructions here: https://www.compose.com/articles/postgresql-tips-installing-the-postgresql-client/ to install it\n")
		os.Exit(1)
	}
}

func CheckVPN() {
	checkVPN, ok := os.LookupEnv("RIDECTL_VPN_CHECK")
	if ok && checkVPN == "false" {
		return
	} else {
		resp, err := http.Head("https://ridectl.s3.us-west-2.amazonaws.com/machinload01.png")
		if err != nil {
			pterm.Error.Println("\n", err)
		}
		if resp.StatusCode != 200 {
			pterm.Error.Println("VPN is not connected")
			os.Exit(1)
		}
	}
}

func CheckTshLogin() {
	if !exec.CheckBinary("tsh") {
		pterm.Info.Println("Tsh cli not found, installing...")
		err := exec.InstallTsh()
		if err != nil {
			pterm.Error.Printf("Error installing tsh : %s\n", err)
			os.Exit(1)
		}
		pterm.Info.Println("Tsh installation completed.")
	}
	// Check if tsh login profile is active or not
	statusArgs := []string{"status"}
	err := exec.ExecuteCommand("tsh", statusArgs, false)
	if err == nil {
		return
	}
	// check if no teleport profile present, ask user to login
	if strings.Contains(err.Error(), "Active profile expired") {
		return
	}
	pterm.Error.Println("No teleport profile found. Refer teleport login command from FAQs:\nhttps://ridecell.quip.com/CILaAnAUnkla/Ridectl-FAQs ")
	os.Exit(1)
}
