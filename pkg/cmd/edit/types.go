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

package edit

import (
	"k8s.io/apimachinery/pkg/runtime"

	secretsv1beta2 "github.com/Ridecell/ridecell-controllers/apis/secrets/v1beta2"
	hacksecretsv1beta2 "github.com/Ridecell/ridectl/pkg/apis/secrets/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Manifest []*Object

type Object struct {
	// The original text as parsed by NewYAMLOrJSONDecoder.
	Raw []byte
	// The original object as decoded by UniversalDeserializer.
	Object runtime.Object
	Meta   metav1.Object

	// Tracking for the various stages of encryption and decryption.
	OrigEnc  *secretsv1beta2.EncryptedSecret
	OrigDec  *hacksecretsv1beta2.DecryptedSecret
	AfterDec *hacksecretsv1beta2.DecryptedSecret
	AfterEnc *secretsv1beta2.EncryptedSecret
	Kind     string
	Data     map[string]string

	// The KMS KeyId used for this object, if known. If nil, it might be a new
	// object.
	KeyId string

	// Byte coordinates for areas of the raw text we need to edit when re-serializing.
	KindLoc TextLocation
	DataLoc TextLocation
	KeyLocs []KeysLocation
}

type TextLocation struct {
	Start int
	End   int
}

type KeysLocation struct {
	TextLocation
	Key string
}

// NotificationsSpec defines notificiations settings for this instance.
type NotificationsSpec struct {
	// list of slack channels for notifications
	PublicSlackChannels []string `yaml:"publicSlackChannels,omitempty"`
	// GithubActions webhook for triggering Regression Test suite on QA tenants - DEVOPS-2348
	GithubactionsRegressionWebhook bool `yaml:"githubactionsRegressionWebhook,omitempty"`
}

// AutoScalingSpec defines configuration and settings for AutoScaling.
type AutoScalingSpec struct {
	Enabled bool `yaml:"enabled,omitempty"`
	// Threshold
	Threshold int32 `yaml:"threshold,omitempty"`
	// Minimum replica count
	MinReplicaCount int32 `yaml:"minReplicaCount,omitempty"`
	// Maximum replica count
	MaxReplicaCount int32 `yaml:"maxReplicaCount,omitempty"`
}

// CelerySpec defines configuration and settings for Celery.
type CelerySpec struct {
	// auto scaling for celeryd
	AutoScaling AutoScalingSpec `yaml:"autoscaling,omitempty"`
	// Setting for --concurrency.
	Concurrency int `yaml:"concurrency,omitempty"`
	// Setting for --pool.
	Pool string `yaml:"pool,omitempty"`
	Replicas *int32 `yaml:"replicas,omitempty"`
}

// WebSpec defines configuration and settings for Web.
type WebSpec struct {
	Replicas *int32 `yaml:"replicas,omitempty"`
}

// DaphneSpec defines configuration and settings for Daphne.
type DaphneSpec struct {
	Replicas *int32 `yaml:"replicas,omitempty"`
}

// CeleryRedBeatSpec defines configuration and settings for CeleryRedBeat.
type CeleryRedBeatSpec struct {
	Replicas *int32 `yaml:"replicas,omitempty"`
}

// ChannelWorkerSpec defines configuration and settings for ChannelWorker.
type ChannelWorkerSpec struct {
	Replicas *int32 `yaml:"replicas,omitempty"`
}

// StaticSpec defines configuration and settings for Static.
type StaticSpec struct {
	Replicas *int32 `yaml:"replicas,omitempty"`
}

// RedisSpec defines resource configuration for redis deployment.
type RedisSpec struct {
	// Setting for tuning redis memory request/limit in MB.
	RAM int `yaml:"ram,omitempty"`
	Replicas *int32 `yaml:"replicas,omitempty"`
	// Type defines the type of redis deployment, self-hosted or elasticache
	Type string `yaml:"type,omitempty"`
	// InstanceType defines the type of redis instance, such as cache.t2.micro
	InstanceType string `yaml:"instanceType,omitempty"`
}
type Resource struct {
	CPU    string `yaml:"cpu,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}
type ResourceRequirements struct {
	// Limits describes the maximum amount of compute resources allowed.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
	Limits Resource `yaml:"limits,omitempty" protobuf:"bytes,1,rep,name=limits,casttype=ResourceList,castkey=ResourceName"`
	// Requests describes the minimum amount of compute resources required.
	// If Requests is omitted for a container, it defaults to Limits if that is explicitly specified,
	// otherwise to an implementation-defined value.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
	Requests Resource `yaml:"requests,omitempty" protobuf:"bytes,2,rep,name=requests,casttype=ResourceList,castkey=ResourceName"`
}

// DispatchSpec defines configuration and settings for Dispatch.
type DispatchSpec struct {
	Replicas *int32 `yaml:"replicas,omitempty"`
	// Compute Resources required by this container.
	Resources ResourceRequirements `yaml:"resources,omitempty" protobuf:"bytes,8,opt,name=resources"`
	// image version to deploy.
	Version string `yaml:"version"`
}

// BusinessPortalSpec defines configuration and settings for BusinessPortal.
type BusinessPortalSpec struct {
	Replicas *int32 `yaml:"replicas,omitempty"`
	// image version to deploy.
	Version string `yaml:"version"`
}

// TripShareSpec defines configuration and settings for TripShare.
type TripShareSpec struct {
	Replicas *int32 `yaml:"replicas,omitempty"`
	// image version to deploy.
	Version string `yaml:"version"`
}

// HwAuxSpec defines configuration and settings for HwAux.
type HwAuxSpec struct {
	Replicas *int32 `yaml:"replicas,omitempty"`
	// image version to deploy.
	Version string `yaml:"version"`
}

// PulseSpec defines configuration and settings for Pulse.
type PulseSpec struct {
	Replicas *int32 `yaml:"replicas,omitempty"`
	// image version to deploy.
	Version string `yaml:"version"`
}

// KafkaConsumerSpec defines configuration and settings for KafkaConsumer.
type KafkaConsumerSpec struct {
	Replicas *int32 `yaml:"replicas,omitempty"`
}

// CustomerPortalSpec defines configuration and settings for CustomerPortal.
type CustomerPortalSpec struct {
	Replicas *int32 `yaml:"replicas,omitempty"`
	// image version to deploy.
	Version string `yaml:"version"`
}

// BackupSpec defines the configuration of the automatic RDS Snapshot feature.
type BackupSpec struct {
	// The ttl of the created rds snapshot in string form.
	TTL metav1.Duration `yaml:"ttl,omitempty"`
	// whether or not the backup process waits on the snapshot to finish
	SkipWaitUntilReady bool `yaml:"skipWaitUntilReady,omitempty"`
}

// OverridesSpec defines values to be overridden
type OverridesSpec struct {
	// which database instance(rds) to use create database, this will point to RDSInstance of crossplane operator
	DatabaseInstance string `yaml:"databaseInstance,omitempty"`
	DatabaseName string `yaml:"databaseName,omitempty"`
	RabbitMQCluster string `yaml:"rabbitmqCluster,omitempty"`
	Vhostname string `yaml:"vhostname,omitempty"`
	RedisHostname string `yaml:"redisHostname,omitempty"`
}

// SummonPlatformSpec defines the desired state of SummonPlatform
type SummonPlatformSpec struct {
	// Hostname aliases (for vanity purposes)
	Aliases []string `yaml:"aliases,omitempty"`
	// Summon image version to deploy. If this isn't specified, AutoDeploy must be.
	Version string `yaml:"version,omitempty"`
	// Summon-platform.yml configuration options.
	Config interface{} `yaml:"config,omitempty"`
	// Settings for deploy and error notifications.
	Notifications NotificationsSpec `yaml:"notifications,omitempty"`
	// Settings for Optimus integration.
	OptimusBucketName string `yaml:"optimusBucketName,omitempty"`
	// Disable the creation of the dispatcher@ridecell.com superuser.
	NoCreateSuperuser bool `yaml:"noCreateSuperuser,omitempty"`
	// SQS queue setting
	SQSQueue string `yaml:"sqsQueue,omitempty"`
	// Environment setting.
	Environment string `yaml:"environment,omitempty"`
	// Enable NewRelic APM.
	EnableNewRelic bool `yaml:"enableNewRelic,omitempty"`
	// Automated backup settings.
	Backup BackupSpec `yaml:"backup,omitempty"`
	// Migration override settings.
	Overrides OverridesSpec `yaml:"overrides,omitempty"`
	// Celery settings.
	Celery CelerySpec `yaml:"celery,omitempty"`
	// Redis resource settings.
	Redis RedisSpec `yaml:"redis,omitempty"`
	// Web settings.
	Web WebSpec `yaml:"web,omitempty"`
	// Daphne settings.
	Daphne DaphneSpec `yaml:"daphne,omitempty"`
	// CeleryRedBeat settings.
	CeleryRedBeat CeleryRedBeatSpec `yaml:"celeryRedBeat,omitempty"`
	// ChannelWorker settings.
	ChannelWorker ChannelWorkerSpec `yaml:"channelWorker,omitempty"`
	// Static settings.
	Static StaticSpec `yaml:"static,omitempty"`
	// Settings for comp-dispatch.
	Dispatch DispatchSpec `yaml:"dispatch,omitempty"`
	// Settings for comp-business-portal.
	BusinessPortal BusinessPortalSpec `yaml:"businessPortal,omitempty"`
	// Settings for comp-trip-share.
	TripShare TripShareSpec `yaml:"tripShare,omitempty"`
	// Settings for comp-hw-aux.
	HwAux HwAuxSpec `yaml:"hwAux,omitempty"`
	// Settings for comp-pulse.
	Pulse PulseSpec `yaml:"pulse,omitempty"`
	// Settings for comp-customer-portal.
	CustomerPortal CustomerPortalSpec `yaml:"customerPortal,omitempty"`
	// Settings for kafkaconsumer
	KafkaConsumer KafkaConsumerSpec `yaml:"kafkaConsumer,omitempty"`
	// Feature flag to disable the CORE-1540 fixup in case it goes AWOL.
	// To be removed when support for the 1540 fixup is removed in summon.
	NoCore1540Fixup bool `yaml:"noCore1540Fixup,omitempty"`
	// Enable mock car server
	EnableMockCarServer bool `yaml:"enableMockCarServer,omitempty"`
	// If Mock car server enabled, provide tenant hardware type
	// +kubebuilder:validation:Enum=OTAKEYS;MENSA
	MockTenantHardwareType string `yaml:"mockTenantHardwareType,omitempty"`
	// ReadOnly users. This list holds the users that will be granted read-only access to the database.
	// Eg. ["user1", "user2"] will create user1_readonly and user2_readonly users.
	ReadOnlyDbUsers []string `yaml:"readOnlyDbUsers,omitempty"`
}

type SummonPlatform struct {
	metav1.TypeMeta   `yaml:",inline"`
	metav1.ObjectMeta `yaml:"metadata,omitempty"`
	APIVersion        string             `yaml:"apiVersion,omitempty"`
	Spec              SummonPlatformSpec `yaml:"spec,omitempty"`
}
