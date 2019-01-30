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

package kubernetes

import (
	"regexp"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

func GetClient(kubeconfig string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func FindSummonPod(clientset *kubernetes.Clientset, instanceName string, labelSelector string) (*corev1.Pod, error) {
	match := regexp.MustCompile(`^[a-z0-9]+-([a-z]+)$`).FindStringSubmatch(instanceName)
	if match == nil {
		return nil, errors.Errorf("unable to parse instance name %s", instanceName)
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}
	pods, err := clientset.CoreV1().Pods(match[1]).List(listOptions)
	if err != nil {
		return nil, err
	}
	if len(pods.Items) == 0 {
		return nil, errors.Errorf("no pods found for %s", listOptions.LabelSelector)
	}
	// It doesn't generally matter which we pick, so just use the first.
	return &pods.Items[0], nil
}
