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
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
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

func GetDynamicClient(kubeconfig string) (dynamic.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return dynamicClient, nil
}

func FindSummonPod(clientset *kubernetes.Clientset, instanceName string, labelSelector string) (*corev1.Pod, error) {
	match := regexp.MustCompile(`^[a-z0-9]+-([a-z]+)$`).FindStringSubmatch(instanceName)
	if match == nil {
		return nil, errors.Errorf("unable to parse instance name %s", instanceName)
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}
	pods, err := clientset.CoreV1().Pods(fmt.Sprintf("summon-%s", match[1])).List(listOptions)
	if err != nil {
		return nil, err
	}
	if len(pods.Items) == 0 {
		return nil, errors.Errorf("no pods found for %s", listOptions.LabelSelector)
	}
	// It doesn't generally matter which we pick, so just use the first.
	return &pods.Items[0], nil
}

func GetPod(clientset *kubernetes.Clientset, namespace string, podRegex string, labelSelector string) (*corev1.Pod, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}
	pods, err := clientset.CoreV1().Pods(fmt.Sprintf("summon-%s", namespace)).List(listOptions)
	if err != nil {
		return nil, err
	}

	for _, pod := range pods.Items {
		match := regexp.MustCompile(podRegex).Match([]byte(pod.Name))
		if match {
			return &pod, nil
		}
	}
	return nil, errors.New("unable to find pod")
}

func FindSummonObject(dynamicClient dynamic.Interface, instanceName string) (*unstructured.Unstructured, error) {
	match := regexp.MustCompile(`^[a-z0-9]+-([a-z]+)$`).FindStringSubmatch(instanceName)
	if match == nil {
		return &unstructured.Unstructured{}, errors.Errorf("unable to parse instance name %s", instanceName)
	}

	env := strings.Split(instanceName, "-")[1]
	namespace := fmt.Sprintf("summon-%s", env)

	summonObjects, err := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    "summon.ridecell.io",
		Version:  "v1beta1",
		Resource: "summonplatforms",
	}).Namespace(namespace).Get(instanceName, metav1.GetOptions{}, "")
	if err != nil {
		return &unstructured.Unstructured{}, err
	}
	return summonObjects, nil
}

func FindSecret(clientset *kubernetes.Clientset, instanceName string, secretName string) (*corev1.Secret, error) {
	match := regexp.MustCompile(`^[a-z0-9]+-([a-z]+)$`).FindStringSubmatch(instanceName)
	if match == nil {
		return nil, errors.Errorf("unable to parse instance name %s", instanceName)
	}

	secret, err := clientset.CoreV1().Secrets(fmt.Sprintf("summon-%s", match[1])).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return secret, nil
}
