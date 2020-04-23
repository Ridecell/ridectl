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
	"bytes"
	"fmt"
	"html/template"
	"time"

	"github.com/Ridecell/ridectl/pkg/exec"
	"github.com/Ridecell/ridectl/pkg/kubernetes"
	"github.com/pkg/errors"
	"github.com/shurcooL/httpfs/vfsutil"
	"github.com/spf13/cobra"

	appsv1 "k8s.io/api/apps/v1"
)

func init() {
	rootCmd.AddCommand(rollingRestartCmd)
}

var rollingRestartCmd = &cobra.Command{
	Use:   "restart [flags] <cluster_name> <pod_type>",
	Short: "Performs a rolling restart of pods.",
	Long:  `Restarts all pods of a certain type (web|celeryd|etc).`,
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("Cluster name argument is required.")
		}
		if len(args) == 1 {
			return fmt.Errorf("Deployment type argument is required.")
		}
		if len(args) > 2 {
			return fmt.Errorf("Too many arguments")
		}
		return nil
	},
	RunE: func(_ *cobra.Command, args []string) error {
		target, err := kubernetes.ParseSubject(args[0])
		if err != nil {
			return errors.Wrap(err, "not a valid target")
		}
		objectName := fmt.Sprintf("%s-%s", args[0], args[1])

		fetchObject := &kubernetes.KubeObject{
			Top: &appsv1.Deployment{},
		}
		err = kubernetes.GetObject(kubeconfigFlag, objectName, target.Namespace, fetchObject)
		if err != nil {
			return errors.Wrap(err, "unable to find deployment")
		}

		deployment, ok := fetchObject.Top.(*appsv1.Deployment)
		if !ok {
			return errors.New("unable to convert runtime.object to corev1.pod")
		}

		templateData, err := vfsutil.ReadFile(Templates, "rolling_restart.json.tpl")
		if err != nil {
			return errors.Wrap(err, "error reading rolling_restart.json.tpl")
		}
		restartTemplate, err := template.New("rolling_restart.json").Parse(string(templateData))
		if err != nil {
			return errors.Wrap(err, "failed to parse restart template")
		}

		buffer := &bytes.Buffer{}
		err = restartTemplate.Execute(buffer, struct {
			Timestamp string
		}{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		if err != nil {
			return errors.Wrap(err, "unable to execute template")
		}

		fmt.Printf("Initiating rolling restart of pods belonging to %s/%s\n", deployment.Namespace, deployment.Name)

		// Spawn kubectl exec.
		kubectlArgs := []string{"kubectl", "patch", "deployment", "-n", deployment.Namespace, deployment.Name, "--context", fetchObject.Context.Name, "-p", buffer.String()}
		return exec.Exec(kubectlArgs)
	},
}
