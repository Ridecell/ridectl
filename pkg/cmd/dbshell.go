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
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Ridecell/ridectl/pkg/exec"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"

	k8s "github.com/Ridecell/ridectl/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	rootCmd.AddCommand(dbShellCmd)
}

const summon = "summon-"

type Kubeobject struct {
	Object crclient.Object
}

func fetchSecret(channel chan Kubeobject, kubeconfig string, k8scontext *api.Context) {

	fmt.Printf("getting secret in %s\n", k8scontext.Cluster)
	k8sClient, err := k8s.GetClientByContext(kubeconfig, k8scontext)
	if err != nil {
		fmt.Println("\nthis is error in getting k8s client\n", err)
	}
	secret := &corev1.Secret{}
	err = k8sClient.Get(context.Background(), types.NamespacedName{Name: "summontest-dev.postgres-user-password", Namespace: summon + "summontest-dev"}, secret)
	if err != nil {
		fmt.Printf("\nUnable to find secret in %s\n", k8scontext.Cluster)
	}
	channel <- Kubeobject{Object: secret}
	fmt.Println("writing to channel")
}
func getAppropriateSecret(kubeconfig string, contexts map[string]*api.Context) Kubeobject {
	secretchannel := make(chan Kubeobject, len(contexts))
	defer close(secretchannel)
	for _, context := range contexts {
		fmt.Printf("\nrunning fetchsecret in %s\n", context.Cluster)
		go fetchSecret(secretchannel, kubeconfig, context)
	}
	fmt.Printf("\nthis is kubeobject %+v\n", Kubeobject{})
	return Kubeobject{}
}

var dbShellCmd = &cobra.Command{
	Use:   "dbshell [flags] <cluster_name>",
	Short: "Open a database shell on a Summon instance or microservice",
	Long: "Open an interactive PostgreSQL shell for a Summon instance or microservice running on Kubernetes.\n" +
		"For summon instances: dbshell <tenant>-<env>                   -- e.g. ridectl dbshell darwin-qa\n" +
		"For microservices: dbshell svc-<region>-<env>-<microservice>   -- e.g. ridectl dbshell svc-us-master-dispatch",
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("Cluster name argument is required")
		}
		if len(args) > 1 {
			return fmt.Errorf("Too many arguments")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()

		kubeContexts, err := k8s.GetKubeContexts()
		if err != nil {
			fmt.Println("\nthis is error in getting kubecontext\n", err)
		}

		kobj := getAppropriateSecret(*kubeconfig, kubeContexts)
		fmt.Printf("%+v", kobj)
		os.Exit(0)
		k8sClient, err := k8s.GetClientByContext(*kubeconfig, kubeContexts["us-sandbox.kops.ridecell.io"])
		if err != nil {
			fmt.Println("\nthis is error in getting k8s client\n", err)
		}
		secret := &corev1.Secret{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: args[0] + ".postgres-user-password", Namespace: summon + "summontest-dev"}, secret)
		if err != nil {
			fmt.Println("secret not found")
		}

		psqlCmd := []string{"psql", "-h", string(secret.Data["host"]), "-U", string(secret.Data["username"]), string(secret.Data["dbname"])}
		os.Setenv("PGPASSWORD", string(secret.Data["password"]))
		return exec.Exec(psqlCmd)

	},
}
