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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Ridecell/ridectl/pkg/exec"
	"github.com/Ridecell/ridectl/pkg/kubernetes"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

func init() {
	rootCmd.AddCommand(dbShellCmd)
}

var dbShellCmd = &cobra.Command{
	Use:   "dbshell [flags] <cluster_name>",
	Short: "Open a database shell on a Summon instance",
	Long:  `Open an interactive PostgreSQL shell for a Summon instance running on Kubernetes`,
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("Cluster name argument is required")
		}
		if len(args) > 1 {
			return fmt.Errorf("Too many arguments")
		}
		return nil
	},
	RunE: func(_ *cobra.Command, args []string) error {

		fetchObject := &kubernetes.KubeObject{Top: &summonv1beta1.SummonPlatform{}}
		namespace := kubernetes.ParseNamespace(args[0])
		err := kubernetes.GetObject(args[0], namespace, fetchObject)
		if err != nil {
			return err
		}

		summonObject, ok := fetchObject.Top.(*summonv1beta1.SummonPlatform)
		if !ok {
			return errors.New("unable to convert to summonplatform object")
		}
		postgresConnection := summonObject.Status.PostgresConnection

		fetchSecret := &corev1.Secret{}
		err = kubernetes.GetObjectWithClient(fetchObject.Client, postgresConnection.PasswordSecretRef.Name, namespace, fetchSecret)
		if err != nil {
			return err
		}

		tempfile, err := ioutil.TempFile("", "")
		if err != nil {
			return errors.Wrap(err, "failed to create tempfile")
		}
		defer os.Remove(tempfile.Name())

		tempfilepath, err := filepath.Abs(tempfile.Name())
		if err != nil {
			return err
		}

		password := fetchSecret.Data[postgresConnection.PasswordSecretRef.Key]

		// hostname:port:database:username:password
		passwordFileString := fmt.Sprintf("%s:%s:%s:%s:%s", postgresConnection.Host, "*", postgresConnection.Database, postgresConnection.Username, password)
		_, err = tempfile.Write([]byte(passwordFileString))
		if err != nil {
			return errors.Wrap(err, "failed to write password to tempfile")
		}

		psqlCmd := []string{"psql", "-h", postgresConnection.Host, "-U", postgresConnection.Username, postgresConnection.Database}
		os.Setenv("PGPASSFILE", tempfilepath)
		return exec.Exec(psqlCmd)
	},
}
