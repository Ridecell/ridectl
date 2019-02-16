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
	"os"
	"os/exec"
	"time"

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

		clientset, err := kubernetes.GetClient(kubeconfigFlag)
		if err != nil {
			return errors.Wrap(err, "unable to load Kubernetes configuration")
		}
		pod, err := kubernetes.FindSummonPod(clientset, args[0], fmt.Sprintf("app.kubernetes.io/instance=%s-web", args[0]))
		if err != nil {
			return errors.Wrap(err, "unable to find pod")
		}

		// Need to check if our input is a file or not.
		inFile, err := os.Open(args[1])
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		var cmd *exec.Cmd
		if os.IsNotExist(err) {
			// Our arg is not a file, assuming it's for s3
			flavorString, err := getPresignedURL(args[1])
			if err != nil {
				return err
			}
			cmd = genCommand(flavorString, pod)
		} else {
			// Our arg is a file, open it and stream it through stdin into the container
			flavorString := "/dev/stdin"
			cmd = genCommand(flavorString, pod)
			cmd.Stdin = inFile
			defer inFile.Close()
		}

		_, err = cmd.CombinedOutput()
		if err != nil {
			return err
		}
		return nil
	},
}

func getPresignedURL(flavorName string) (string, error) {
	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region: aws.String("us-west-2"),
		},
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		return "", err
	}
	svc := s3.New(sess)
	req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String("ridecell-flavors"),
		Key:    aws.String(fmt.Sprintf("%s.json.bz2", flavorName)),
	})

	urlStr, err := req.Presign(15 * time.Minute)
	if err != nil {
		return "", errors.Wrapf(err, "failed to presign s3 url for object key %s.json.bz2", flavorName)
	}

	return urlStr, nil
}

func genCommand(input string, pod *corev1.Pod) *exec.Cmd {

	cmdArgs := []string{"exec", "-i", "-n", pod.Namespace, pod.Name, "--", "python", "manage.py", "loadflavor", input}
	if eraseDatabaseFlag {
		cmdArgs = append(cmdArgs, "--erase-database")
	}
	cmd := exec.Command("kubectl", cmdArgs...)
	return cmd
}
