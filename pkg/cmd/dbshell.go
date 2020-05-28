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
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

var disabledns bool

func init() {
	dbShellCmd.Flags().BoolVarP(&disabledns, "disabledns", "", false, "(optional) will not use aws dns to resolve postgres host")
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
			return fmt.Errorf("Cluster name argument is required")
		}
		if len(args) > 1 {
			return fmt.Errorf("Too many arguments")
		}
		return nil
	},
	RunE: func(_ *cobra.Command, args []string) error {
		// Determine if we are trying to connect to a microservice or summonplatform db
		target, err := kubernetes.ParseSubject(args[0])
		if err != nil {
			return err
		}
		fetchObject := &kubernetes.KubeObject{Top: &dbv1beta1.PostgresDatabase{}}
		err = kubernetes.GetObject(kubeconfigFlag, target.Name, target.Namespace, fetchObject)
		if err != nil {
			return err
		}

		pgdbObject, ok := fetchObject.Top.(*dbv1beta1.PostgresDatabase)
		if !ok {
			return errors.New("unable to convert to PostgresDatabase object")
		}
		postgresConnection := pgdbObject.Status.Connection
		fetchSecret := &corev1.Secret{}

		err = kubernetes.GetObjectWithClient(fetchObject.Client, postgresConnection.PasswordSecretRef.Name, target.Namespace, fetchSecret)
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

		host := postgresConnection.Host
		// get rds local ip
		if !disabledns {
			fmt.Println("Using aws dns...")
			host, err = resolveAwsRecord(postgresConnection.Host)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
		// hostname:port:database:username:password
		passwordFileString := fmt.Sprintf("%s:%s:%s:%s:%s", host, "*", postgresConnection.Database, postgresConnection.Username, password)
		_, err = tempfile.Write([]byte(passwordFileString))
		if err != nil {
			return errors.Wrap(err, "failed to write password to tempfile")
		}

		psqlCmd := []string{"psql", "-h", host, "-U", postgresConnection.Username, postgresConnection.Database}
		os.Setenv("PGPASSFILE", tempfilepath)
		return exec.Exec(psqlCmd)
	},
}

func resolveAwsRecord(hostname string) (string, error) {
	server := "172.30.2.63"
	m1 := new(dns.Msg)
	m1.Id = dns.Id()
	m1.RecursionDesired = true
	m1.Question = make([]dns.Question, 1)
	m1.Question[0] = dns.Question{dns.Fqdn(hostname), dns.TypeA, dns.ClassINET}
	r, err := dns.Exchange(m1, server+":53000")
	if err != nil {
		errors.Wrap(err, "failed get aws local ip for "+hostname)
	}

	if len(r.Answer) == 0 {
		return "", errors.New("No records return")
	}
	for _, record := range r.Answer {
		if t, ok := record.(*dns.A); ok {
			return t.A.String(), nil
		}
	}
	return "", errors.New("No A record returned")
}
