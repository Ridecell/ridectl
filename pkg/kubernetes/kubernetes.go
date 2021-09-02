/*
Copyright 2021 Ridecell, Inc.
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
	"sync"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	summonv1beta2 "github.com/Ridecell/summon-operator/apis/app/v1beta2"
	appsv1 "k8s.io/api/apps/v1"
)

const namespacePrefix = "summon-"

type Kubeobject struct {
	Object  client.Object
	Context *api.Context
	Client  client.Client
}

type Subject struct {
	Region    string
	Env       string
	Namespace string
	Name      string
	Type      string
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

func fetchContextForObject(channel chan Kubeobject, cluster *api.Context, crclient client.Client, subject Subject, wg *sync.WaitGroup) {

	defer wg.Done()

	var objectName string
	if subject.Type == "summon" {
		summonObj := &summonv1beta2.SummonPlatform{}
		err := crclient.Get(context.TODO(), types.NamespacedName{Name: subject.Name, Namespace: subject.Namespace}, summonObj)
		if err != nil {
			return
		}
		if err == nil {
			channel <- Kubeobject{Object: summonObj, Context: cluster, Client: crclient}
			return
		}

	} else if subject.Type == "microservice" {
		objectName = fmt.Sprintf("%s-svc-%s-web", subject.Env, subject.Namespace)

		deploymentObj := &appsv1.Deployment{}
		err := crclient.Get(context.Background(), types.NamespacedName{Name: objectName, Namespace: subject.Namespace}, deploymentObj)
		if err != nil {
			fmt.Printf("Instance %s not found in %s\n", subject.Name, cluster.Cluster)
			return
		}
		if err == nil {
			channel <- Kubeobject{Client: crclient, Context: cluster}
		}
	}

}

func GetAppropriateObjectWithContext(kubeconfig string, instance string, subject Subject) Kubeobject {

	contexts, err := getKubeContexts()
	if err != nil {
		fmt.Println("Error getting kubecontexts", err)
		return Kubeobject{}
	}

	k8sClients := make(map[string]client.Client)
	for _, context := range contexts {
		k8sClient, err := getClientByContext(kubeconfig, context)
		if err != nil {
			continue
		}
		k8sClients[context.Cluster] = k8sClient
	}
	// Initialize a wait group
	var wg sync.WaitGroup
	wg.Add(len(k8sClients))

	objChannel := make(chan Kubeobject, len(k8sClients))
	defer close(objChannel)

	for cluster, client := range k8sClients {
		go fetchContextForObject(objChannel, contexts[cluster], client, subject, &wg)
	}
	// Block until all of my goroutines have processed their issues.
	wg.Wait()
	if len(objChannel) < 1 {
		return Kubeobject{}
	}
	return <-objChannel
}

// Parses the instance and returns an array of strings denoting: [region, env, subject, namespace]
func ParseSubject(instanceName string) (Subject, error) {
	var subject Subject
	microservice := regexp.MustCompile(`svc-(\w+)-(\w+)-(.+)`)
	// Summon instance name can't start with a digit since it is used with a Service -- needs a valid DNS name.
	summon := regexp.MustCompile(`([a-z][a-z0-9]+)-([a-z]+)`)

	svcMatch := microservice.MatchString(instanceName)
	if svcMatch {
		fields := microservice.FindStringSubmatch(instanceName)
		subject.Name = fields[0]
		subject.Region = fields[1]
		subject.Env = fields[2]
		subject.Namespace = fields[3]
		subject.Type = "microservice"
		return subject, nil
	}

	sMatch := summon.MatchString(instanceName)
	if sMatch {
		fields := summon.FindStringSubmatch(instanceName)
		// summon instances can only parse out name, env and namespace
		subject.Name = fields[0] // want summon name to keep env as well
		subject.Env = fields[2]
		subject.Namespace = namespacePrefix + subject.Name
		subject.Type = "summon"
		return subject, nil
	}
	// Nothing matched, return empty with error
	return subject, fmt.Errorf("could not parse out information from %s", instanceName)
}
