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
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Ridecell/ridecell-operator/pkg/apis"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	corev1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

const namespacePrefix = "summon-"

type KubeObject struct {
	Top     runtime.Object
	Client  client.Client
	Context *kubeContext
}

type kubeContext struct {
	Name    string
	Context *api.Context
}

func init() {
	err := apis.AddToScheme(scheme.Scheme)
	if err != nil {
		// Panic cause this should not happen.
		panic(err)
	}
}

func listPodsWithContext(kubeconfig string, contextObj *kubeContext, listOptions *client.ListOptions, podList chan *KubeObject) {
	contextClient, err := getClientByContext(kubeconfig, contextObj.Context)
	if err != nil {
		// User may have an invalid context that causes this to fail. Just return nil and continue.
		podList <- nil
		return
	}

	fetchPodList := &corev1.PodList{}
	err = contextClient.List(context.Background(), listOptions, fetchPodList)
	if err != nil {
		podList <- nil
		return
	}

	if len(fetchPodList.Items) == 0 {
		podList <- nil
		return
	}

	newKubeObject := &KubeObject{
		Top:     fetchPodList,
		Client:  contextClient,
		Context: contextObj,
	}
	podList <- newKubeObject
}

func GetPod(kubeconfig string, nameRegex *string, labelSelector *string, namespace string, fetchObject *KubeObject) error {
	listOptions := &client.ListOptions{
		Namespace: namespace,
	}

	if labelSelector != nil {
		listOptions.SetLabelSelector(*labelSelector)
	}

	kubeContexts, err := getKubeContexts()
	if err != nil {
		return err
	}

	ch := make(chan *KubeObject, len(kubeContexts))
	for contextName, contextObj := range kubeContexts {
		kubeContextObj := &kubeContext{
			Name:    contextName,
			Context: contextObj,
		}
		go listPodsWithContext(kubeconfig, kubeContextObj, listOptions, ch)
	}

	tempObject, err := getChannelOutput(len(kubeContexts), ch)
	if err != nil {
		return err
	}
	fetchObject.Client = tempObject.Client
	fetchObject.Context = tempObject.Context

	podList, ok := tempObject.Top.(*corev1.PodList)
	if !ok {
		return errors.New("unable to convert top object to podlist")
	}

	if nameRegex != nil {
		for _, pod := range podList.Items {
			match := regexp.MustCompile(*nameRegex).Match([]byte(pod.Name))
			if match {
				fetchObject.Top = &pod
				return nil
			}
		}
		return errors.New("unable to find pod matching regex")
	}
	fetchObject.Top = &podList.Items[0]
	return nil
}

func GetObject(kubeconfig string, name string, namespace string, fetchObject *KubeObject) error {
	kubeContexts, err := getKubeContexts()
	if err != nil {
		return err
	}

	ch := make(chan *KubeObject, len(kubeContexts))
	for contextName, contextObj := range kubeContexts {
		kubeContextObj := &kubeContext{
			Name:    contextName,
			Context: contextObj,
		}
		go getObjectWithContext(kubeconfig, fetchObject.Top, name, namespace, kubeContextObj, ch)
	}

	tempObject, err := getChannelOutput(len(kubeContexts), ch)
	if err != nil {
		return err
	}
	fetchObject.Top = tempObject.Top
	fetchObject.Client = tempObject.Client
	fetchObject.Context = tempObject.Context
	return nil
}

func GetObjectWithClient(contextClient client.Client, name string, namespace string, runtimeObj runtime.Object) error {
	err := contextClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, runtimeObj)
	if err != nil {
		return err
	}
	return nil
}

func getObjectWithContext(kubeconfig string, runtimeObj runtime.Object, name string, namespace string, contextObj *kubeContext, fetchObject chan *KubeObject) {
	contextClient, err := getClientByContext(kubeconfig, contextObj.Context)
	if err != nil {
		// User may have an invalid context that causes this to fail. Just return nil and continue.
		fetchObject <- nil
		return
	}

	err = contextClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, runtimeObj)
	if err != nil {
		fetchObject <- nil
		return
	}

	newKubeObject := &KubeObject{
		Top:     runtimeObj,
		Client:  contextClient,
		Context: contextObj,
	}
	fetchObject <- newKubeObject
}

func getKubeContexts() (map[string]*api.Context, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{})
	rawConfig, err := config.RawConfig()
	if err != nil {
		return nil, err
	}
	return rawConfig.Contexts, nil
}

func getClientByContext(kubeconfig string, kubeContext *api.Context) (client.Client, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = kubeconfig
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{Context: *kubeContext})
	cfg, err := config.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get client with context")
	}

	mapper, err := apiutil.NewDiscoveryRESTMapper(cfg)
	if err != nil {
		return nil, err
	}

	client, err := client.New(cfg, client.Options{Scheme: scheme.Scheme, Mapper: mapper})
	if err != nil {
		return nil, err
	}

	return client, nil
}

func ParseNamespace(instanceName string) string {
	env := strings.Split(instanceName, "-")[1]
	namespace := fmt.Sprintf("%s%s", namespacePrefix, env)
	return namespace
}

func getChannelOutput(maxFails int, ch chan *KubeObject) (*KubeObject, error) {
	fails := 0
	for {
		select {
		case s := <-ch:
			if s == nil {
				fails++
				if fails >= maxFails {
					return nil, errors.New("unable to locate object")
				}
			} else {
				return s, nil
			}
		}
	}
}
