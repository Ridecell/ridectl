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
	"crypto/md5"
	"encoding/hex"
	"flag"
	"io"
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
	if _, installed := exec.CheckBinary("kubectl"); !installed {
		pterm.Error.Printf("kubectl is not installed. Follow the instructions here: https://kubernetes.io/docs/tasks/tools/#kubectl to install it\n")
		os.Exit(1)
	}
}

func CheckPsql() {
	if _, installed := exec.CheckBinary("psql"); !installed {
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

func installTsh() {
	err := exec.InstallTsh()
	if err != nil {
		pterm.Error.Printf("Error installing tsh : %s\n", err)
		os.Exit(1)
	}
	pterm.Info.Println("Tsh installation completed.")
}

func CheckTshLogin() {
	binPath, installed := exec.CheckBinary("tsh")
	if !installed {
		pterm.Info.Println("Tsh cli not found, installing using sudo...")
		installTsh()
	}

	//Generate MD5 hash of installed tsh binary
	f, err := os.Open(binPath)
 	if err != nil {
 		pterm.Error.Printf("Error opening tsh : %s\n", err)
		os.Exit(1)
 	}
 	defer f.Close()

 	hash := md5.New()
 	_, err = io.Copy(hash, f)
 	if err != nil {
 		pterm.Error.Printf("Error generating hash for tsh : %s\n", err)
		os.Exit(1)
 	}
	// Check if tsh binary's md5 is same; if not, install tsh
	if hex.EncodeToString(hash.Sum(nil)) != exec.GetTshMd5Hash() {
		pterm.Info.Println("Tsh version not matched, re-installing using sudo...")
		installTsh()
	}

	// Check if tsh login profile is active or not
	statusArgs := []string{"status"}
	err = exec.ExecuteCommand("tsh", statusArgs, false)
	if err == nil {
		return
	}
	// check if no teleport profile present, ask user to login
	if strings.Contains(err.Error(), "Active profile expired") {
		return
	}
	pterm.Error.Println("No teleport profile found. Refer teleport login command from FAQs:\nhttps://ridecell.quip.com/CILaAnAUnkla/Ridectl-FAQs#temp:C:ZZZabcdcead11c941ccbb5ad29b3 ")
	os.Exit(1)
}
