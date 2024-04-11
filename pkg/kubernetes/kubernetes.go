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
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	summonv1beta2 "github.com/Ridecell/summon-operator/apis/app/v1beta2"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
)

const namespacePrefix = "summon-"

type Kubeobject struct {
	Object  client.Object
	Context string
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

	var cfg *rest.Config
	var err error
	if kubeconfig == "" {
		// empty kubeconfig, use in-cluster config
		cfg, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	} else {

		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		loadingRules.ExplicitPath = kubeconfig

		config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loadingRules,
			&clientcmd.ConfigOverrides{Context: *kubeContext})
		cfg, err = config.ClientConfig()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get client with context")
		}

		// host port check does not apply to github runners using ridectl
		checkTSH := os.Getenv("RIDECTL_TSH_CHECK")

		// Return error to skip searching non-ridecell hosts
		if checkTSH != "false" && !strings.Contains(cfg.Host, "teleport") {
			return nil, errors.New("hostname did not match, ignoring context")
		}
	}
	// Set high timeout, since user has to login if their teleport login is expired.
	cfg.Timeout = time.Minute * 3
	httpClient, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, err
	}
	mapper, err := apiutil.NewDynamicRESTMapper(cfg, httpClient)
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

func fetchContextForObject(channel chan Kubeobject, clusterName string, cluster *api.Context, crclient client.Client, subject Subject, wg *sync.WaitGroup) {

	defer wg.Done()

	var objectName string
	if subject.Type == "summon" {
		summonObj := &summonv1beta2.SummonPlatform{}
		pterm.Info.Printf("Checking instance in %s\n", clusterName)
		err := crclient.Get(context.TODO(), types.NamespacedName{Name: subject.Name, Namespace: subject.Namespace}, summonObj)
		if err != nil {
			pterm.Warning.Printf("%s in %s\n", err.Error(), clusterName)
			return
		}

		if err == nil {
			channel <- Kubeobject{Object: summonObj, Context: clusterName, Client: crclient}
			return
		}

	} else if subject.Type == "microservice" {
		objectName = fmt.Sprintf("%s-svc-%s-web", subject.Env, subject.Namespace)

		deploymentObj := &appsv1.Deployment{}
		pterm.Info.Printf("Checking instance in %+v\n", clusterName)
		err := crclient.Get(context.Background(), types.NamespacedName{Name: objectName, Namespace: subject.Namespace}, deploymentObj)
		if err != nil {
			pterm.Warning.Printf("%s in %s\n", err.Error(), clusterName)
			return
		}
		// This makes sure we are returning the correct context.
		// In the case of microservices, the deployment name is same for all clusters
		if err == nil && deploymentObj.Labels["region"] == subject.Region {
			channel <- Kubeobject{Client: crclient, Context: clusterName}
		} else {
			return
		}
	} else if subject.Type == "job" {
		jobObj := &batchv1.Job{}

		pterm.Info.Printf(" Checking job in %s\n", cluster.Cluster)
		err := crclient.Get(context.Background(), types.NamespacedName{Name: subject.Name, Namespace: subject.Namespace}, jobObj)
		if err != nil {
			pterm.Warning.Printf("%s in %s\n", err.Error(), cluster.Cluster)
			return
		}
		channel <- Kubeobject{Object: jobObj, Client: crclient, Context: clusterName}
		return
	}
}

func GetAppropriateObjectWithContext(kubeconfig string, instance string, subject Subject, inCluster bool) (Kubeobject, error) {

	if inCluster {
		var kubeObj Kubeobject
		k8sclient, err := getClientByContext("", nil)
		if err != nil {
			return kubeObj, errors.Wrap(err, ": Error getting incluster client")
		}
		kubeObj = Kubeobject{
			Client: k8sclient,
		}
		return kubeObj, nil
	}

	contexts, err := getKubeContexts()
	if err != nil {
		return Kubeobject{}, errors.Wrap(err, ": Error getting kubecontexts")
	}

	k8sClients := make(map[string]client.Client)
	for clusterName, context := range contexts {
		if !validCluster(clusterName, subject.Env) {
			continue
		}
		k8sClient, err := getClientByContext(kubeconfig, context)
		if err != nil {
			continue
		}
		k8sClients[clusterName] = k8sClient
	}

	if len(k8sClients) < 1 {
		return Kubeobject{}, errors.New("No valid cluster was found")
	}

	// Initialize a wait group
	var wg sync.WaitGroup
	wg.Add(len(k8sClients))

	objChannel := make(chan Kubeobject, len(k8sClients))
	defer close(objChannel)

	for cluster, client := range k8sClients {
		go fetchContextForObject(objChannel, cluster, contexts[cluster], client, subject, &wg)
	}
	// Block until all of my goroutines have processed their issues.
	wg.Wait()
	if len(objChannel) < 1 {
		return Kubeobject{}, nil
	}
	return <-objChannel, nil
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
	return subject, fmt.Errorf("could not parse out information from %s.", instanceName)
}

// Return true only if given Environment is present in target cluster
func validCluster(clusterName string, env string) bool {
	if env == "prod" || env == "uat" {
		return strings.Contains(clusterName, "prod.kops")
	}
	return !strings.Contains(clusterName, "prod.kops")
}

// Returns Pod container's status if it is ready
func IsContainerReady(status *v1.PodStatus) bool {
	if status != nil && status.Conditions != nil {
		for i := range status.Conditions {
			if status.Conditions[i].Type == v1.ContainersReady {
				return status.Conditions[i].Status == v1.ConditionTrue
			}
		}
	}
	return false
}
