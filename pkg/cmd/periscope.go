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
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/Ridecell/ridectl/pkg/kubernetes"

	dbv1beta2 "github.com/Ridecell/ridecell-controllers/apis/db/v1beta2"
	corev1 "k8s.io/api/core/v1"
)

func init() {
	rootCmd.AddCommand(periscopeCmd)
}

var periscopeCmd = &cobra.Command{
	Use:   "periscope <cluster_name>",
	Short: "Dumps Periscope data to setup database.",
	Long:  "Dumps relevant Periscope data required to setup databases on the periscopdata web interface.",
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
		dbname := args[0]
		env := strings.Split(args[0], "-")[1]
		target, err := kubernetes.ParseSubject(args[0])
		if err != nil {
			return errors.Wrap(err, "not a valid target")
		}
		var secretName string

		if env == "prod" {
			secretName = fmt.Sprintf("%s-periscope.postgres-user-password", args[0])
		} else {
			secretName = fmt.Sprintf("summon-%s-periscope.postgres-user-password", env)
		}

		fetchObject := &kubernetes.KubeObject{Top: &corev1.Secret{}}
		err = kubernetes.GetObject(kubeconfigFlag, secretName, target.Namespace, fetchObject)
		if err != nil {
			return errors.Wrap(err, "unable to find secret")
		}
		secret, ok := fetchObject.Top.(*corev1.Secret)
		if !ok {
			return errors.New("unable to convert to secret object")
		}

		fetchObject = &kubernetes.KubeObject{Top: &dbv1beta2.PostgresDatabase{}}
		err = kubernetes.GetObject(kubeconfigFlag, dbname, target.Namespace, fetchObject)
		if err != nil {
			return errors.Wrap(err, "unable to find PostgresDatabase info")
		}
		database, ok := fetchObject.Top.(*dbv1beta2.PostgresDatabase)
		if !ok {
			return errors.New("unable to get PostgresDatabase object")
		}
		//fet postgresuser
		fetchObject = &kubernetes.KubeObject{Top: &dbv1beta2.PostgresUser{}}
		err = kubernetes.GetObject(kubeconfigFlag, dbname+".postgres-user-password", target.Namespace, fetchObject)
		if err != nil {
			return errors.Wrap(err, "unable to find PostgresUser info")
		}

		databaseUser, ok := fetchObject.Top.(*dbv1beta2.PostgresUser)
		if !ok {
			return errors.New("unable to get PostgresUser object")
		}

		fmt.Printf("Periscope Data\n================\n")
		fmt.Printf("Database Type: Postgres\n") // Hard code-y
		fmt.Printf("Database Host: %s\n", secret.Data["endpoint"])
		fmt.Printf("Database Port: %d\n", secret.Data["port"])
		fmt.Printf("Database Name: %s\n", database.Spec.DatabaseName)
		fmt.Printf("Database Username: %s\n", databaseUser.Spec.Username)
		fmt.Printf("Database Password: %s\n\n", string(secret.Data["password"]))
		return nil
	},
}
