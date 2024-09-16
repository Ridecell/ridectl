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
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/Ridecell/ridectl/pkg/exec"
	"github.com/Ridecell/ridectl/pkg/kubernetes"
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
		pterm.Error.Printf("psql is not installed locally. Follow the instructions here: https://www.timescale.com/blog/how-to-install-psql-on-mac-ubuntu-debian-windows to install it locally\n")
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
	checkTSH, ok := os.LookupEnv("RIDECTL_TSH_CHECK")
	if ok && checkTSH == "false" {
		return
	}
	err := exec.InstallOrUpgradeTsh()
	if err != nil {
		pterm.Error.Printf("Error while installing or upgrading tsh : %s\n", err)
		os.Exit(1)
	}

	// Check if tsh login profile is active or not
	statusArgs := []string{"status"}
	err = exec.ExecuteCommand("tsh", statusArgs, false)
	if err == nil {
		// Execute tsh kube login command for automatically populate kubeconfig
		populateKubeConfig()
		return
	}
	// check if no teleport profile present, ask user to login
	if strings.Contains(err.Error(), "Active profile expired") {
		return
	}
	pterm.Error.Println("No teleport profile found. Refer teleport login command from FAQs:\nhttps://docs.google.com/document/d/1v6lbH4NgN6rHBHpELWrcQ4CyqwVeSgeP/preview#heading=h.kyqd59381iiz ")
	os.Exit(1)
}

func populateKubeConfig() {
	// Check last modified date of /tmp/ridectl-kube.lock file
	filename := "/tmp/ridectl-kube.lock"
	tmpFile, err := os.Stat(filename)
	if err != nil {
		// Executing tsh kube login
		kubeLoginArgs := []string{"kube", "login", "--all"}
		err = exec.ExecuteCommand("tsh", kubeLoginArgs, false)
		if err != nil {
			pterm.Warning.Printf("Error configuring kubernetes contexts: %s\n", err)
			return
		}
		err = os.WriteFile(filename, []byte(""), 0644)
		if err != nil {
			pterm.Warning.Printf("Error creating temp file: %s\n", err)
		}
		return
	}
	// Check if file is modified in last 7 days
	now := time.Now()
	if now.Sub(tmpFile.ModTime()).Hours()/24 > 7 {
		kubeLoginArgs := []string{"kube", "login", "--all"}
		err = exec.ExecuteCommand("tsh", kubeLoginArgs, false)
		if err != nil {
			pterm.Warning.Printf("Error configuring kubernetes contexts: %s\n", err)
			return
		}
		if err := os.Chtimes(filename, now, now); err != nil {
			pterm.Warning.Printf("Error updating temp file stat: %s\n", err)
		}
		return
	}
}

func DoesInstanceExist(name string, inCluster bool) (kubernetes.Subject, kubernetes.Kubeobject, bool) {
	kubeconfig := GetKubeconfig()
	target, err := kubernetes.ParseSubject(name)
	var kubeObj kubernetes.Kubeobject
	if err != nil {
		pterm.Error.Println(err, "It's not a valid Summonplatform or Microservice")
		return target, kubeObj, false
	}

	// inCluster from root.go is set via ridectl cmd args, defaulting to false.
	kubeObj, err = kubernetes.GetAppropriateObjectWithContext(*kubeconfig, name, target, inCluster)
	if err != nil {
		pterm.Error.Printf("%s", err.Error())
		return target, kubeObj, false
	}
	if reflect.DeepEqual(kubeObj, kubernetes.Kubeobject{}) {
		pterm.Error.Printf("No instance found [%s]. Double check the following:\n"+
			"- Instance name is correct\n"+
			"- You have the required access in Infra-Auth\n"+
			"For more details and help with the above, see: https://docs.google.com/document/d/1v6lbH4NgN6rHBHpELWrcQ4CyqwVeSgeP/preview#heading=h.xq8mwj7wt9h1\n", name)
		return target, kubeObj, false
	}

	return target, kubeObj, true
}

func GetAnnouncementMessage() string {
	resp, err := http.Get("https://ridectl.s3.us-west-2.amazonaws.com/ridectl-announcement-banner.txt")
	if err == nil && resp.StatusCode == 200 {
		content, err := io.ReadAll(resp.Body)
		if err == nil {
			return string(content)
		}
	}
	defer resp.Body.Close()
	return ""
}
