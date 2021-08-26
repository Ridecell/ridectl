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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/Ridecell/ridectl/pkg/exec"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/util/homedir"

	kubernetes "github.com/Ridecell/ridectl/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
)

func init() {
	rootCmd.AddCommand(dbShellCmd)
}

var dbShellCmd = &cobra.Command{
	Use:   "dbshell [flags] <cluster_name>",
	Short: "Open a database shell on a Summon instance or microservice",
	Long: "Open an interactive PostgreSQL shell for a Summon instance or microservice running on Kubernetes.\n" +
		"For summon instances: dbshell <tenant>-<env>                   -- e.g. ridectl dbshell darwin-qa\n" +
		"For microservices: dbshell svc-<region>-<env>-<microservice>   -- e.g. ridectl dbshell svc-us-master-dispatch",
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("cluster name argument is required")
		}
		if len(args) > 1 {
			return fmt.Errorf("too many arguments")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		//ctx := context.Background()
		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()
		target, err := kubernetes.ParseSubject(args[0])
		if err != nil {
			return errors.Wrap(err, "not a valid target")
		}
		kubeObj := kubernetes.GetAppropriateObjectWithContext(*kubeconfig, args[0], target, nil)
		if reflect.DeepEqual(kubeObj, kubernetes.Kubeobject{}) {
			return fmt.Errorf("no instance found")
		}

		dbSecret := kubeObj.Object.(*corev1.Secret)
		psqlCmd := []string{"psql", "-h", string(dbSecret.Data["host"]), "-U", string(dbSecret.Data["username"]), string(dbSecret.Data["dbname"])}
		os.Setenv("PGPASSWORD", string(dbSecret.Data["password"]))
		return exec.Exec(psqlCmd)
	},
}
