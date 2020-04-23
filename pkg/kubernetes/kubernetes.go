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
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/Ridecell/ridecell-operator/pkg/apis"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
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

	// Return error to skip searching non-ridecell hosts
	if !strings.Contains(cfg.Host, ".kops.ridecell.io") {
		return nil, errors.New("hostname did not match, ignoring context")
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

// Parses the instance and returns an array of strings denoting: [region, env, subject, namespace]
func ParseSubject(instanceName string) []string {

	microservice := regexp.MustCompile(`svc-(\w+)-(\w+)-(.+)`)
	summon := regexp.MustCompile(`(\w+)-(\w+)`)

	svcMatch := microservice.MatchString(instanceName)
	if svcMatch {
		fields := microservice.FindStringSubmatch(instanceName)

		return []string{fields[1], fields[2], fields[3], fields[3]}
	}

	sMatch := summon.MatchString(instanceName)
	if sMatch {
		fields := summon.FindStringSubmatch(instanceName)

		// summon instances can only parse out subject and namespace
		return []string{"", "", fields[1], namespacePrefix + fields[2]}
	}

	// Nothing matched, return empty
	return []string{}
}

func ParseNamespace(instanceName string) string {
	parsed := ParseSubject(instanceName)
	return parsed[3]
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

func listSummonPlatformWithContext(kubeconfig string, contextObj *kubeContext, listOptions *client.ListOptions, summonList chan *KubeObject) {
	contextClient, err := getClientByContext(kubeconfig, contextObj.Context)
	if err != nil {
		// User may have an invalid context that causes this to fail. Just return nil and continue.
		summonList <- nil
		return
	}

	fetchSummonList := &summonv1beta1.SummonPlatformList{}
	err = contextClient.List(context.Background(), listOptions, fetchSummonList)
	if err != nil {
		summonList <- nil
		return
	}

	if len(fetchSummonList.Items) == 0 {
		summonList <- nil
		return
	}

	newKubeObject := &KubeObject{
		Top:     fetchSummonList,
		Client:  contextClient,
		Context: contextObj,
	}
	summonList <- newKubeObject
}

func ListSummonPlatforms(kubeconfig string, nameregex string, namespace string) (summonv1beta1.SummonPlatformList, error) {
	summonPlatformLists := &summonv1beta1.SummonPlatformList{}
	listOptions := &client.ListOptions{
		Namespace: namespace,
	}

	kubeContexts, err := getKubeContexts()
	if err != nil {
		return *summonPlatformLists, err
	}

	ch := make(chan *KubeObject, len(kubeContexts))
	for contextName, contextObj := range kubeContexts {
		kubeContextObj := &kubeContext{
			Name:    contextName,
			Context: contextObj,
		}
		go listSummonPlatformWithContext(kubeconfig, kubeContextObj, listOptions, ch)
	}

	// go through each cluster and grab summon platforms
	for i := 0; i < len(kubeContexts); i++ {
		tempObject := <-ch
		if tempObject == nil {
			continue
		}

		summonPlatformList, ok := tempObject.Top.(*summonv1beta1.SummonPlatformList)
		if !ok {
			return *summonPlatformList, errors.New("unable to convert top object to summonPlatformList")
		}

		// if searching for a single tenant, just return a summonPlatformList with that single tenant
		if nameregex != "" {
			for _, summonplatform := range summonPlatformList.Items {
				match := regexp.MustCompile(nameregex).Match([]byte(summonplatform.Name))
				if match {
					summonPlatformLists.Items = append(summonPlatformLists.Items, summonplatform)
					return *summonPlatformLists, nil
				}
			}
		} else {
			summonPlatformLists.Items = append(summonPlatformLists.Items, summonPlatformList.Items...)
		}
	}

	// if we went through all the clusters and still haven't found a match, return error
	if nameregex != "" {
		return *summonPlatformLists, errors.New("unable to find %s" + nameregex + "\n")
	}
	// Sort the list in alphabetical order
	sort.Slice(summonPlatformLists.Items, func(i, j int) bool {
		return summonPlatformLists.Items[i].Name < summonPlatformLists.Items[j].Name
	})

	return *summonPlatformLists, nil
}
