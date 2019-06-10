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
	"compress/bzip2"
	"fmt"
	"os"
	"os/exec"

	"github.com/Ridecell/ridectl/pkg/kubernetes"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	corev1 "k8s.io/api/core/v1"
)

func init() {
	rootCmd.AddCommand(loadflavorCmd)
}

var eraseDatabaseFlag bool

func init() {
	rootCmd.Flags().BoolVar(&eraseDatabaseFlag, "erase-database", false, "Erases database before loading flavor data.")
}

var loadflavorCmd = &cobra.Command{
	Use:   "loadflavor [flags] <cluster_name> <filepath|flavor_name>",
	Short: "Loads a database flavor into a Summon container",
	Long:  `Loads specified database flavor into a Summon database from s3 or local file.`,
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("Cluster name and flavor arguments are required")
		}
		if len(args) > 2 {
			return fmt.Errorf("Too many arguments")
		}
		return nil
	},
	RunE: func(_ *cobra.Command, args []string) error {
		namespace := kubernetes.ParseNamespace(args[0])
		labelSelector := fmt.Sprintf("app.kubernetes.io/instance=%s-web", args[0])

		fetchObject := &kubernetes.KubeObject{}
		err := kubernetes.GetPod(kubeconfigFlag, nil, &labelSelector, namespace, fetchObject)
		if err != nil {
			return errors.Wrap(err, "unable to find pod")
		}

		contextName := fetchObject.Context.Name

		pod, ok := fetchObject.Top.(*corev1.Pod)
		if !ok {
			return errors.New("unable to convert runtime.object to corev1.pod")
		}

		cmdArgs := []string{"exec", "-i", "-n", pod.Namespace, pod.Name, "--context", contextName, "--", "python", "manage.py", "loadflavor", "/dev/stdin"}
		if eraseDatabaseFlag {
			cmdArgs = append(cmdArgs, "--erase-database")
		}
		cmd := exec.Command("kubectl", cmdArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Need to check if our input is a file or not.
		inFile, err := os.Open(args[1])
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		cmd.Stdin = inFile

		if os.IsNotExist(err) {
			// Our arg is not a file, assume it's an s3 key
			sess, err := session.NewSession()
			if err != nil {
				return err
			}
			s3svc := s3.New(sess, aws.NewConfig().WithRegion("us-west-2"))
			object, err := s3svc.GetObject(&s3.GetObjectInput{
				Bucket: aws.String("ridecell-flavors"),
				Key:    aws.String(args[1]),
			})
			if err != nil {
				return err
			}
			// Decompress bzip2
			cmd.Stdin = bzip2.NewReader(object.Body)
		}

		err = cmd.Run()
		if err != nil {
			return err
		}

		return nil
	},
}
