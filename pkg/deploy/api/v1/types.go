package v1

import (
	"k8s.io/kubernetes/pkg/api/unversioned"
	kapi "k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/util/intstr"
)

// DeploymentPhase describes the possible states a deployment can be in.
type DeploymentPhase string

const (
	// DeploymentPhaseNew means the deployment has been accepted but not yet acted upon.
	DeploymentPhaseNew DeploymentPhase = "New"
	// DeploymentPhasePending means the deployment been handed over to a deployment strategy,
	// but the strategy has not yet declared the deployment to be running.
	DeploymentPhasePending DeploymentPhase = "Pending"
	// DeploymentPhaseRunning means the deployment strategy has reported the deployment as
	// being in-progress.
	DeploymentPhaseRunning DeploymentPhase = "Running"
	// DeploymentPhaseComplete means the deployment finished without an error.
	DeploymentPhaseComplete DeploymentPhase = "Complete"
	// DeploymentPhaseFailed means the deployment finished with an error.
	DeploymentPhaseFailed DeploymentPhase = "Failed"
)

// DeploymentStrategy describes how to perform a deployment.
type DeploymentStrategy struct {
	// Type is the name of a deployment strategy.
	Type DeploymentStrategyType `json:"type,omitempty" protobuf:"bytes,1,opt,name=type,casttype=DeploymentStrategyType"`

	// CustomParams are the input to the Custom deployment strategy.
	CustomParams *CustomDeploymentStrategyParams `json:"customParams,omitempty" protobuf:"bytes,2,opt,name=customParams"`
	// RecreateParams are the input to the Recreate deployment strategy.
	RecreateParams *RecreateDeploymentStrategyParams `json:"recreateParams,omitempty" protobuf:"bytes,3,opt,name=recreateParams"`
	// RollingParams are the input to the Rolling deployment strategy.
	RollingParams *RollingDeploymentStrategyParams `json:"rollingParams,omitempty" protobuf:"bytes,4,opt,name=rollingParams"`

	// Resources contains resource requirements to execute the deployment and any hooks
	Resources kapi.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,5,opt,name=resources"`
	// Labels is a set of key, value pairs added to custom deployer and lifecycle pre/post hook pods.
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,6,rep,name=labels"`
	// Annotations is a set of key, value pairs added to custom deployer and lifecycle pre/post hook pods.
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,7,rep,name=annotations"`
}

// DeploymentStrategyType refers to a specific DeploymentStrategy implementation.
type DeploymentStrategyType string

const (
	// DeploymentStrategyTypeRecreate is a simple strategy suitable as a default.
	DeploymentStrategyTypeRecreate DeploymentStrategyType = "Recreate"
	// DeploymentStrategyTypeCustom is a user defined strategy.
	DeploymentStrategyTypeCustom DeploymentStrategyType = "Custom"
	// DeploymentStrategyTypeRolling uses the Kubernetes RollingUpdater.
	DeploymentStrategyTypeRolling DeploymentStrategyType = "Rolling"

	// DeploymentStrategyTypeMarathon is to deploy to marathon site
	DeploymentStrategyTypeMarathon DeploymentStrategyType = "Marathon"
)

// CustomDeploymentStrategyParams are the input to the Custom deployment strategy.
type CustomDeploymentStrategyParams struct {
	// Image specifies a Docker image which can carry out a deployment.
	Image string `json:"image,omitempty" protobuf:"bytes,1,opt,name=image"`
	// Environment holds the environment which will be given to the container for Image.
	Environment []kapi.EnvVar `json:"environment,omitempty" protobuf:"bytes,2,rep,name=environment"`
	// Command is optional and overrides CMD in the container Image.
	Command []string `json:"command,omitempty" protobuf:"bytes,3,rep,name=command"`
}

// RecreateDeploymentStrategyParams are the input to the Recreate deployment
// strategy.
type RecreateDeploymentStrategyParams struct {
	// TimeoutSeconds is the time to wait for updates before giving up. If the
	// value is nil, a default will be used.
	TimeoutSeconds *int64 `json:"timeoutSeconds,omitempty" protobuf:"varint,1,opt,name=timeoutSeconds"`
	// Pre is a lifecycle hook which is executed before the strategy manipulates
	// the deployment. All LifecycleHookFailurePolicy values are supported.
	Pre *LifecycleHook `json:"pre,omitempty" protobuf:"bytes,2,opt,name=pre"`
	// Mid is a lifecycle hook which is executed while the deployment is scaled down to zero before the first new
	// pod is created. All LifecycleHookFailurePolicy values are supported.
	Mid *LifecycleHook `json:"mid,omitempty" protobuf:"bytes,3,opt,name=mid"`
	// Post is a lifecycle hook which is executed after the strategy has
	// finished all deployment logic. All LifecycleHookFailurePolicy values are supported.
	Post *LifecycleHook `json:"post,omitempty" protobuf:"bytes,4,opt,name=post"`
}

// LifecycleHook defines a specific deployment lifecycle action. Only one type of action may be specified at any time.
type LifecycleHook struct {
	// FailurePolicy specifies what action to take if the hook fails.
	FailurePolicy LifecycleHookFailurePolicy `json:"failurePolicy" protobuf:"bytes,1,opt,name=failurePolicy,casttype=LifecycleHookFailurePolicy"`

	// ExecNewPod specifies the options for a lifecycle hook backed by a pod.
	ExecNewPod *ExecNewPodHook `json:"execNewPod,omitempty" protobuf:"bytes,2,opt,name=execNewPod"`

	// TagImages instructs the deployer to tag the current image referenced under a container onto an image stream tag.
	TagImages []TagImageHook `json:"tagImages,omitempty" protobuf:"bytes,3,rep,name=tagImages"`
}

// LifecycleHookFailurePolicy describes possibles actions to take if a hook fails.
type LifecycleHookFailurePolicy string

const (
	// LifecycleHookFailurePolicyRetry means retry the hook until it succeeds.
	LifecycleHookFailurePolicyRetry LifecycleHookFailurePolicy = "Retry"
	// LifecycleHookFailurePolicyAbort means abort the deployment (if possible).
	LifecycleHookFailurePolicyAbort LifecycleHookFailurePolicy = "Abort"
	// LifecycleHookFailurePolicyIgnore means ignore failure and continue the deployment.
	LifecycleHookFailurePolicyIgnore LifecycleHookFailurePolicy = "Ignore"
)

// ExecNewPodHook is a hook implementation which runs a command in a new pod
// based on the specified container which is assumed to be part of the
// deployment template.
type ExecNewPodHook struct {
	// Command is the action command and its arguments.
	Command []string `json:"command" protobuf:"bytes,1,rep,name=command"`
	// Env is a set of environment variables to supply to the hook pod's container.
	Env []kapi.EnvVar `json:"env,omitempty" protobuf:"bytes,2,rep,name=env"`
	// ContainerName is the name of a container in the deployment pod template
	// whose Docker image will be used for the hook pod's container.
	ContainerName string `json:"containerName" protobuf:"bytes,3,opt,name=containerName"`
	// Volumes is a list of named volumes from the pod template which should be
	// copied to the hook pod. Volumes names not found in pod spec are ignored.
	// An empty list means no volumes will be copied.
	Volumes []string `json:"volumes,omitempty" protobuf:"bytes,4,rep,name=volumes"`
}

// TagImageHook is a request to tag the image in a particular container onto an ImageStreamTag.
type TagImageHook struct {
	// ContainerName is the name of a container in the deployment config whose image value will be used as the source of the tag. If there is only a single
	// container this value will be defaulted to the name of that container.
	ContainerName string `json:"containerName" protobuf:"bytes,1,opt,name=containerName"`
	// To is the target ImageStreamTag to set the container's image onto.
	To kapi.ObjectReference `json:"to" protobuf:"bytes,2,opt,name=to"`
}

// RollingDeploymentStrategyParams are the input to the Rolling deployment
// strategy.
type RollingDeploymentStrategyParams struct {
	// UpdatePeriodSeconds is the time to wait between individual pod updates.
	// If the value is nil, a default will be used.
	UpdatePeriodSeconds *int64 `json:"updatePeriodSeconds,omitempty" protobuf:"varint,1,opt,name=updatePeriodSeconds"`
	// IntervalSeconds is the time to wait between polling deployment status
	// after update. If the value is nil, a default will be used.
	IntervalSeconds *int64 `json:"intervalSeconds,omitempty" protobuf:"varint,2,opt,name=intervalSeconds"`
	// TimeoutSeconds is the time to wait for updates before giving up. If the
	// value is nil, a default will be used.
	TimeoutSeconds *int64 `json:"timeoutSeconds,omitempty" protobuf:"varint,3,opt,name=timeoutSeconds"`
	// MaxUnavailable is the maximum number of pods that can be unavailable
	// during the update. Value can be an absolute number (ex: 5) or a
	// percentage of total pods at the start of update (ex: 10%). Absolute
	// number is calculated from percentage by rounding up.
	//
	// This cannot be 0 if MaxSurge is 0. By default, 25% is used.
	//
	// Example: when this is set to 30%, the old RC can be scaled down by 30%
	// immediately when the rolling update starts. Once new pods are ready, old
	// RC can be scaled down further, followed by scaling up the new RC,
	// ensuring that at least 70% of original number of pods are available at
	// all times during the update.
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty" protobuf:"bytes,4,opt,name=maxUnavailable"`
	// MaxSurge is the maximum number of pods that can be scheduled above the
	// original number of pods. Value can be an absolute number (ex: 5) or a
	// percentage of total pods at the start of the update (ex: 10%). Absolute
	// number is calculated from percentage by rounding up.
	//
	// This cannot be 0 if MaxUnavailable is 0. By default, 25% is used.
	//
	// Example: when this is set to 30%, the new RC can be scaled up by 30%
	// immediately when the rolling update starts. Once old pods have been
	// killed, new RC can be scaled up further, ensuring that total number of
	// pods running at any time during the update is atmost 130% of original
	// pods.
	MaxSurge *intstr.IntOrString `json:"maxSurge,omitempty" protobuf:"bytes,5,opt,name=maxSurge"`
	// UpdatePercent is the percentage of replicas to scale up or down each
	// interval. If nil, one replica will be scaled up and down each interval.
	// If negative, the scale order will be down/up instead of up/down.
	// DEPRECATED: Use MaxUnavailable/MaxSurge instead.
	UpdatePercent *int32 `json:"updatePercent,omitempty" protobuf:"varint,6,opt,name=updatePercent"`
	// Pre is a lifecycle hook which is executed before the deployment process
	// begins. All LifecycleHookFailurePolicy values are supported.
	Pre *LifecycleHook `json:"pre,omitempty" protobuf:"bytes,7,opt,name=pre"`
	// Post is a lifecycle hook which is executed after the strategy has
	// finished all deployment logic. The LifecycleHookFailurePolicyAbort policy
	// is NOT supported.
	Post *LifecycleHook `json:"post,omitempty" protobuf:"bytes,8,opt,name=post"`
}

// These constants represent keys used for correlating objects related to deployments.
const (
	// DeploymentConfigAnnotation is an annotation name used to correlate a deployment with the
	// DeploymentConfig on which the deployment is based.
	DeploymentConfigAnnotation = "openshift.io/deployment-config.name"
	// DeploymentAnnotation is an annotation on a deployer Pod. The annotation value is the name
	// of the deployment (a ReplicationController) on which the deployer Pod acts.
	DeploymentAnnotation = "openshift.io/deployment.name"
	// DeploymentPodAnnotation is an annotation on a deployment (a ReplicationController). The
	// annotation value is the name of the deployer Pod which will act upon the ReplicationController
	// to implement the deployment behavior.
	DeploymentPodAnnotation = "openshift.io/deployer-pod.name"
	// DeploymentPodTypeLabel is a label with which contains a type of deployment pod.
	DeploymentPodTypeLabel = "openshift.io/deployer-pod.type"
	// DeployerPodForDeploymentLabel is a label which groups pods related to a
	// deployment. The value is a deployment name. The deployer pod and hook pods
	// created by the internal strategies will have this label. Custom
	// strategies can apply this label to any pods they create, enabling
	// platform-provided cancellation and garbage collection support.
	DeployerPodForDeploymentLabel = "openshift.io/deployer-pod-for.name"
	// DeploymentPhaseAnnotation is an annotation name used to retrieve the DeploymentPhase of
	// a deployment.
	DeploymentPhaseAnnotation = "openshift.io/deployment.phase"
	// DeploymentEncodedConfigAnnotation is an annotation name used to retrieve specific encoded
	// DeploymentConfig on which a given deployment is based.
	DeploymentEncodedConfigAnnotation = "openshift.io/encoded-deployment-config"
	// DeploymentVersionAnnotation is an annotation on a deployment (a ReplicationController). The
	// annotation value is the LatestVersion value of the DeploymentConfig which was the basis for
	// the deployment.
	DeploymentVersionAnnotation = "openshift.io/deployment-config.latest-version"
	// DeploymentLabel is the name of a label used to correlate a deployment with the Pod created
	// to execute the deployment logic.
	// TODO: This is a workaround for upstream's lack of annotation support on PodTemplate. Once
	// annotations are available on PodTemplate, audit this constant with the goal of removing it.
	DeploymentLabel = "deployment"
	// DeploymentConfigLabel is the name of a label used to correlate a deployment with the
	// DeploymentConfigs on which the deployment is based.
	DeploymentConfigLabel = "deploymentconfig"
	// DeploymentStatusReasonAnnotation represents the reason for deployment being in a given state
	// Used for specifying the reason for cancellation or failure of a deployment
	DeploymentStatusReasonAnnotation = "openshift.io/deployment.status-reason"
	// DeploymentCancelledAnnotation indicates that the deployment has been cancelled
	// The annotation value does not matter and its mere presence indicates cancellation
	DeploymentCancelledAnnotation = "openshift.io/deployment.cancelled"
	// DeploymentInstantiatedAnnotation indicates that the deployment has been instantiated.
	// The annotation value does not matter and its mere presence indicates instantiation.
	DeploymentInstantiatedAnnotation = "openshift.io/deployment.instantiated"

	// DeploymentMarathonRetryAnnotation indicates that the marathon deployment has been
	// asked to retry and the annotation value is the deployment config version
	DeploymentMarathonRetryAnnotation = "openshift.io/deployment.marathon.retry"
	// DeploymentMarathonScaleAnnotation indicates that the marathon deployment has been
	// asked to scale up/down and the annotation value is the deployment config version
	DeploymentMarathonScaleAnnotation = "openshift.io/deployment.marathon.scale"
	// DeploymentMarathonReconcileAnnotation indicates that the marathon deployment has been
	// asked to reconciile and the annotation value is the deployment config version
	DeploymentMarathonReconcileAnnotation = "openshift.io/deployment.marathon.reconcile"
)

// +genclient=true

// DeploymentConfig represents a configuration for a single deployment (represented as a
// ReplicationController). It also contains details about changes which resulted in the current
// state of the DeploymentConfig. Each change to the DeploymentConfig which should result in
// a new deployment results in an increment of LatestVersion.
type DeploymentConfig struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard object's metadata.
	kapi.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec represents a desired deployment state and how to deploy to it.
	Spec DeploymentConfigSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`

	// Status represents the current deployment state.
	Status DeploymentConfigStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

// DeploymentTriggerPolicies is a list of policies where nil values and different from empty arrays.
// +protobuf.nullable=true
type DeploymentTriggerPolicies []DeploymentTriggerPolicy

// DeploymentConfigSpec represents the desired state of the deployment.
type DeploymentConfigSpec struct {
	// Strategy describes how a deployment is executed.
	Strategy DeploymentStrategy `json:"strategy" protobuf:"bytes,1,opt,name=strategy"`

	// MinReadySeconds is the minimum number of seconds for which a newly created pod should
	// be ready without any of its container crashing, for it to be considered available.
	// Defaults to 0 (pod will be considered available as soon as it is ready)
	MinReadySeconds int32 `json:"minReadySeconds,omitempty" protobuf:"varint,9,opt,name=minReadySeconds"`

	// Triggers determine how updates to a DeploymentConfig result in new deployments. If no triggers
	// are defined, a new deployment can only occur as a result of an explicit client update to the
	// DeploymentConfig with a new LatestVersion. If null, defaults to having a config change trigger.
	Triggers DeploymentTriggerPolicies `json:"triggers" protobuf:"bytes,2,rep,name=triggers"`

	// Replicas is the number of desired replicas.
	Replicas int32 `json:"replicas" protobuf:"varint,3,opt,name=replicas"`

	// RevisionHistoryLimit is the number of old ReplicationControllers to retain to allow for rollbacks.
	// This field is a pointer to allow for differentiation between an explicit zero and not specified.
	RevisionHistoryLimit *int32 `json:"revisionHistoryLimit,omitempty" protobuf:"varint,4,opt,name=revisionHistoryLimit"`

	// Test ensures that this deployment config will have zero replicas except while a deployment is running. This allows the
	// deployment config to be used as a continuous deployment test - triggering on images, running the deployment, and then succeeding
	// or failing. Post strategy hooks and After actions can be used to integrate successful deployment with an action.
	Test bool `json:"test" protobuf:"varint,5,opt,name=test"`

	// Paused indicates that the deployment config is paused resulting in no new deployments on template
	// changes or changes in the template caused by other triggers.
	Paused bool `json:"paused,omitempty" protobuf:"varint,6,opt,name=paused"`

	// Selector is a label query over pods that should match the Replicas count.
	Selector map[string]string `json:"selector,omitempty" protobuf:"bytes,7,rep,name=selector"`

	// Template is the object that describes the pod that will be created if
	// insufficient replicas are detected.
	Template *kapi.PodTemplateSpec `json:"template,omitempty" protobuf:"bytes,8,opt,name=template"`

	// Site is the identifier that specifies the site where deployment will be conducted
	Site string `json:"site,omitempty" protobuf:"bytes,10,opt,name=site"`

	// MarathonAppTemplate is the object that describes the application that will be created
	// by mesos Marathon scheduler
	MarathonAppTemplate *MarathonApplication `json:"marathonAppTemplate,omitempty" protobuf:"bytes,11,opt,name=marathonAppTemplate"`
}

// DeploymentConfigStatus represents the current deployment state.
type DeploymentConfigStatus struct {
	// LatestVersion is used to determine whether the current deployment associated with a deployment
	// config is out of sync.
	LatestVersion int64 `json:"latestVersion,omitempty" protobuf:"varint,1,opt,name=latestVersion"`
	// ObservedGeneration is the most recent generation observed by the deployment config controller.
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,2,opt,name=observedGeneration"`
	// Replicas is the total number of pods targeted by this deployment config.
	Replicas int32 `json:"replicas,omitempty" protobuf:"varint,3,opt,name=replicas"`
	// UpdatedReplicas is the total number of non-terminated pods targeted by this deployment config
	// that have the desired template spec.
	UpdatedReplicas int32 `json:"updatedReplicas,omitempty" protobuf:"varint,4,opt,name=updatedReplicas"`
	// AvailableReplicas is the total number of available pods targeted by this deployment config.
	AvailableReplicas int32 `json:"availableReplicas,omitempty" protobuf:"varint,5,opt,name=availableReplicas"`
	// UnavailableReplicas is the total number of unavailable pods targeted by this deployment config.
	UnavailableReplicas int32 `json:"unavailableReplicas,omitempty" protobuf:"varint,6,opt,name=unavailableReplicas"`
	// Details are the reasons for the update to this deployment config.
	// This could be based on a change made by the user or caused by an automatic trigger
	Details *DeploymentDetails `json:"details,omitempty" protobuf:"bytes,7,opt,name=details"`
}

// DeploymentTriggerPolicy describes a policy for a single trigger that results in a new deployment.
type DeploymentTriggerPolicy struct {
	// Type of the trigger
	Type DeploymentTriggerType `json:"type,omitempty" protobuf:"bytes,1,opt,name=type,casttype=DeploymentTriggerType"`
	// ImageChangeParams represents the parameters for the ImageChange trigger.
	ImageChangeParams *DeploymentTriggerImageChangeParams `json:"imageChangeParams,omitempty" protobuf:"bytes,2,opt,name=imageChangeParams"`
}

// DeploymentTriggerType refers to a specific DeploymentTriggerPolicy implementation.
type DeploymentTriggerType string

const (
	// DeploymentTriggerOnImageChange will create new deployments in response to updated tags from
	// a Docker image repository.
	DeploymentTriggerOnImageChange DeploymentTriggerType = "ImageChange"
	// DeploymentTriggerOnConfigChange will create new deployments in response to changes to
	// the ControllerTemplate of a DeploymentConfig.
	DeploymentTriggerOnConfigChange DeploymentTriggerType = "ConfigChange"
)

// DeploymentTriggerImageChangeParams represents the parameters to the ImageChange trigger.
type DeploymentTriggerImageChangeParams struct {
	// Automatic means that the detection of a new tag value should result in an image update
	// inside the pod template. Deployment configs that haven't been deployed yet will always
	// have their images updated. Deployment configs that have been deployed at least once, will
	// have their images updated only if this is set to true.
	Automatic bool `json:"automatic,omitempty" protobuf:"varint,1,opt,name=automatic"`
	// ContainerNames is used to restrict tag updates to the specified set of container names in a pod.
	ContainerNames []string `json:"containerNames,omitempty" protobuf:"bytes,2,rep,name=containerNames"`
	// From is a reference to an image stream tag to watch for changes. From.Name is the only
	// required subfield - if From.Namespace is blank, the namespace of the current deployment
	// trigger will be used.
	From kapi.ObjectReference `json:"from" protobuf:"bytes,3,opt,name=from"`
	// LastTriggeredImage is the last image to be triggered.
	LastTriggeredImage string `json:"lastTriggeredImage,omitempty" protobuf:"bytes,4,opt,name=lastTriggeredImage"`
}

// DeploymentDetails captures information about the causes of a deployment.
type DeploymentDetails struct {
	// Message is the user specified change message, if this deployment was triggered manually by the user
	Message string `json:"message,omitempty" protobuf:"bytes,1,opt,name=message"`
	// Causes are extended data associated with all the causes for creating a new deployment
	Causes []DeploymentCause `json:"causes" protobuf:"bytes,2,rep,name=causes"`
}

// DeploymentCause captures information about a particular cause of a deployment.
type DeploymentCause struct {
	// Type of the trigger that resulted in the creation of a new deployment
	Type DeploymentTriggerType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=DeploymentTriggerType"`
	// ImageTrigger contains the image trigger details, if this trigger was fired based on an image change
	ImageTrigger *DeploymentCauseImageTrigger `json:"imageTrigger,omitempty" protobuf:"bytes,2,opt,name=imageTrigger"`
}

// DeploymentCauseImageTrigger represents details about the cause of a deployment originating
// from an image change trigger
type DeploymentCauseImageTrigger struct {
	// From is a reference to the changed object which triggered a deployment. The field may have
	// the kinds DockerImage, ImageStreamTag, or ImageStreamImage.
	From kapi.ObjectReference `json:"from" protobuf:"bytes,1,opt,name=from"`
}

// DeploymentConfigList is a collection of deployment configs.
type DeploymentConfigList struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard object's metadata.
	unversioned.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of deployment configs
	Items []DeploymentConfig `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// DeploymentConfigRollback provides the input to rollback generation.
type DeploymentConfigRollback struct {
	unversioned.TypeMeta `json:",inline"`
	// Name of the deployment config that will be rolled back.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// UpdatedAnnotations is a set of new annotations that will be added in the deployment config.
	UpdatedAnnotations map[string]string `json:"updatedAnnotations,omitempty" protobuf:"bytes,2,rep,name=updatedAnnotations"`
	// Spec defines the options to rollback generation.
	Spec DeploymentConfigRollbackSpec `json:"spec" protobuf:"bytes,3,opt,name=spec"`
}

// DeploymentConfigRollbackSpec represents the options for rollback generation.
type DeploymentConfigRollbackSpec struct {
	// From points to a ReplicationController which is a deployment.
	From kapi.ObjectReference `json:"from" protobuf:"bytes,1,opt,name=from"`
	// Revision to rollback to. If set to 0, rollback to the last revision.
	Revision int64 `json:"revision,omitempty" protobuf:"varint,2,opt,name=revision"`
	// IncludeTriggers specifies whether to include config Triggers.
	IncludeTriggers bool `json:"includeTriggers" protobuf:"varint,3,opt,name=includeTriggers"`
	// IncludeTemplate specifies whether to include the PodTemplateSpec.
	IncludeTemplate bool `json:"includeTemplate" protobuf:"varint,4,opt,name=includeTemplate"`
	// IncludeReplicationMeta specifies whether to include the replica count and selector.
	IncludeReplicationMeta bool `json:"includeReplicationMeta" protobuf:"varint,5,opt,name=includeReplicationMeta"`
	// IncludeStrategy specifies whether to include the deployment Strategy.
	IncludeStrategy bool `json:"includeStrategy" protobuf:"varint,6,opt,name=includeStrategy"`
}

// DeploymentLog represents the logs for a deployment
type DeploymentLog struct {
	unversioned.TypeMeta `json:",inline"`
}

// DeploymentLogOptions is the REST options for a deployment log
type DeploymentLogOptions struct {
	unversioned.TypeMeta `json:",inline"`

	// The container for which to stream logs. Defaults to only container if there is one container in the pod.
	Container string `json:"container,omitempty" protobuf:"bytes,1,opt,name=container"`
	// Follow if true indicates that the build log should be streamed until
	// the build terminates.
	Follow bool `json:"follow,omitempty" protobuf:"varint,2,opt,name=follow"`
	// Return previous deployment logs. Defaults to false.
	Previous bool `json:"previous,omitempty" protobuf:"varint,3,opt,name=previous"`
	// A relative time in seconds before the current time from which to show logs. If this value
	// precedes the time a pod was started, only logs since the pod start will be returned.
	// If this value is in the future, no logs will be returned.
	// Only one of sinceSeconds or sinceTime may be specified.
	SinceSeconds *int64 `json:"sinceSeconds,omitempty" protobuf:"varint,4,opt,name=sinceSeconds"`
	// An RFC3339 timestamp from which to show logs. If this value
	// precedes the time a pod was started, only logs since the pod start will be returned.
	// If this value is in the future, no logs will be returned.
	// Only one of sinceSeconds or sinceTime may be specified.
	SinceTime *unversioned.Time `json:"sinceTime,omitempty" protobuf:"bytes,5,opt,name=sinceTime"`
	// If true, add an RFC3339 or RFC3339Nano timestamp at the beginning of every line
	// of log output. Defaults to false.
	Timestamps bool `json:"timestamps,omitempty" protobuf:"varint,6,opt,name=timestamps"`
	// If set, the number of lines from the end of the logs to show. If not specified,
	// logs are shown from the creation of the container or sinceSeconds or sinceTime
	TailLines *int64 `json:"tailLines,omitempty" protobuf:"varint,7,opt,name=tailLines"`
	// If set, the number of bytes to read from the server before terminating the
	// log output. This may not display a complete final line of logging, and may return
	// slightly more or slightly less than the specified limit.
	LimitBytes *int64 `json:"limitBytes,omitempty" protobuf:"varint,8,opt,name=limitBytes"`

	// NoWait if true causes the call to return immediately even if the deployment
	// is not available yet. Otherwise the server will wait until the deployment has started.
	// TODO: Fix the tag to 'noWait' in v2
	NoWait bool `json:"nowait,omitempty" protobuf:"varint,9,opt,name=nowait"`

	// Version of the deployment for which to view logs.
	Version *int64 `json:"version,omitempty" protobuf:"varint,10,opt,name=version"`
}

// Constraint is the container placement constraint for scheduling an application in marathon
type MarathonConstraint struct {
	// Valid constraint operators are one of ["UNIQUE", "CLUSTER", "GROUP_BY"].
	Constraint []string `json:"constraint,omitempty" protobuf:"bytes,1,rep,name=constraint"`
}

// Application is the definition for an application in marathon
type MarathonApplication struct {
	// Unique identifier for the app consisting of a series of names separated by slashes.
	ID string `json:"id,omitempty" protobuf:"bytes,1,opt,name=id"`

	// The command that is executed.
	Cmd *string `json:"cmd,omitempty" protobuf:"bytes,2,opt,name=cmd"`

	// An array of strings that represents an alternative mode of specifying the command to run.
	Args []string `json:"args,omitempty" protobuf:"bytes,3,rep,name=args"`

	// Valid constraint operators are one of ["UNIQUE", "CLUSTER", "GROUP_BY"].
	Constraints []MarathonConstraint `json:"constraints,omitempty" protobuf:"bytes,4,rep,name=constraints"`

	// Container is the definition for a container type in marathon
	// Additional data passed to the containerizer on application launch. These consist of a type, zero or more volumes, and additional type-specific options. Volumes and type are optional (the default type is DOCKER).
	Container *MarathonContainer `json:"container,omitempty" protobuf:"bytes,5,opt,name=container"`

	// The number of CPU`s this application needs per instance.
	CPUs *float64 `json:"cpus,omitempty" protobuf:"bytes,6,opt,name=cpus"`

	// The number of DISK`s this application needs per instance.
	Disk *float64 `json:"disk,omitempty" protobuf:"bytes,7,opt,name=disk"`

	// Key value pairs that get added to the environment variables of the process to start.
	Env map[string]string `json:"env,omitempty" protobuf:"bytes,8,opt,name=env"`

	// The executor to use to launch this application.
	Executor *string `json:"executor,omitempty" protobuf:"bytes,9,opt,name=executor"`

	// An array of checks to be performed on running tasks to determine if they are operating as expected. Health checks begin immediately upon task launch.
	HealthChecks []MarathonHealthCheck `json:"healthChecks,omitempty" protobuf:"bytes,10,rep,name=healthChecks"`

	// The amount of memory in MB that is needed for the application per instance.
	Mem *float64 `json:"mem,omitempty" protobuf:"bytes,11,opt,name=mem"`

	// Deprecated . Use portDefinitions instead.
	Ports []int32 `json:"ports" protobuf:"bytes,12,rep,name=ports"`

	// Normally, the host ports of your tasks are automatically assigned. This corresponds to the requirePorts value false which is the default.
	// If you need more control and want to specify your host ports in advance, you can set requirePorts to true. This way the ports you have specified are used as host ports. That also means that Marathon can schedule the associated tasks only on hosts that have the specified ports available.
	RequirePorts *bool `json:"requirePorts,omitempty" protobuf:"varint,13,opt,name=requirePorts"`

	// Configures exponential backoff behavior when launching potentially sick apps. This prevents sandboxes associated with consecutively failing tasks from filling up the hard disk on Mesos slaves. The backoff period is multiplied by the factor for each consecutive failure until it reaches maxLaunchDelaySeconds. This applies also to tasks that are killed due to failing too many health checks.
	// backoff seconds when failure happens
	BackoffSeconds *float64 `json:"backoffSeconds,omitempty" protobuf:"bytes,14,opt,name=backoffSeconds"`

	// backoff factor used to multiplied by backoff seconds
	BackoffFactor *float64 `json:"backoffFactor,omitempty" protobuf:"bytes,15,opt,name=backoffFactor"`

	// Max launch delay seconds when launching potentially sick apps
	MaxLaunchDelaySeconds *float64 `json:"maxLaunchDelaySeconds,omitempty" protobuf:"bytes,16,opt,name=maxLaunchDelaySeconds"`

	// A list of services upon which this application depends. An order is derived from the dependencies for performing start/stop and upgrade of the application. For example, an application /a relies on the services /b which itself relies on /c. To start all 3 applications, first /c is started than /b than /a.
	Dependencies []string `json:"dependencies" protobuf:"bytes,17,rep,name=dependencies"`

	// User to launch the application container
	User string `json:"user,omitempty" protobuf:"bytes,18,opt,name=user"`

	// During an upgrade all instances of an application get replaced by a new version.
	UpgradeStrategy *MarathonUpgradeStrategy `json:"upgradeStrategy,omitempty" protobuf:"bytes,19,opt,name=upgradeStrategy"`

	// Since v0.15.0: Deprecated . Use fetch instead.
	Uris []string `json:"uris" protobuf:"bytes,20,rep,name=uris"`

	// Attaching metadata to apps can be useful to expose additional information to other services, so we added the ability to place labels on apps (for example, you could label apps "staging" and "production" to mark services by their position in the pipeline).
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,21,opt,name=labels"`

	// Optional. A list of resource roles. Marathon considers only resource offers with roles in this list for launching tasks of this app. If you do not specify this, Marathon considers all resource offers with roles that have been configured by the --default_accepted_resource_roles command line flag. If no --default_accepted_resource_roles was given on startup, Marathon considers all resource offers.
	AcceptedResourceRoles []string `json:"acceptedResourceRoles,omitempty" protobuf:"bytes,22,rep,name=acceptedResourceRoles"`

	// The list of URIs to fetch before the task starts.
	Fetch []MarathonFetch `json:"fetch" protobuf:"bytes,23,rep,name=fetch"`
}

// Container is the definition for a container type in marathon
type MarathonContainer struct {
	//  container types, currelty "docker"/"mesos" supported
	Type string `json:"type,omitempty" protobuf:"bytes,1,opt,name=type"`

	// Docker is the docker definition from a marathon application
	Docker *MarathonDocker `json:"docker,omitempty" protobuf:"bytes,2,opt,name=docker"`

	// Volume is the docker volume details associated to the container
	Volumes []MarathonVolume `json:"volumes,omitempty" protobuf:"bytes,3,rep,name=volumes"`
}

// Docker is the docker definition from a marathon application
type MarathonDocker struct {
	// ForcePullImage Flag force Docker to pull the image before launching each task, by default false.
	ForcePullImage *bool `json:"forcePullImage,omitempty" protobuf:"varint,1,opt,name=forcePullImage"`

	// Image name of the docker container
	Image string `json:"image,omitempty" protobuf:"bytes,2,opt,name=image"`

	// Network mode, currently support "bridge"/"host"/"none"
	Network string `json:"network,omitempty" protobuf:"bytes,3,opt,name=network"`

	// The parameters object allows users to supply arbitrary command-line options for the docker run command executed by the Mesos containerizer.
	Parameters []MarathonParameters `json:"parameters,omitempty" protobuf:"bytes,4,rep,name=parameters"`

	// PortMapping is the portmapping structure between container and mesos
	PortMappings []MarathonPortMapping `json:"portMappings,omitempty" protobuf:"bytes,5,rep,name=portMappings"`

	// Privileged flag allows users to run containers in privileged mode. This flag is false by default.
	Privileged *bool `json:"privileged,omitempty" protobuf:"varint,6,opt,name=privileged"`
}

// Volume is the docker volume details associated to the container
type MarathonVolume struct {
	// container path refers to the volume path the application accesses inside of the container.
	ContainerPath string `json:"containerPath,omitempty" protobuf:"bytes,1,opt,name=containerPath"`

	// host path refers to the local host volume path to be mounted by the container.
	HostPath string `json:"hostPath,omitempty" protobuf:"bytes,2,opt,name=hostPath"`

	// Read/Write mode of the mounted volume inside of the container. "R"/"W"/"RW"
	Mode string `json:"mode,omitempty" protobuf:"bytes,3,opt,name=mode"`
}

// PortMapping is the portmapping structure between container and mesos
type MarathonPortMapping struct {
	// container port refers to the port the application listens to inside of the container.
	ContainerPort int32 `json:"containerPort,omitempty" protobuf:"varint,1,opt,name=containerPort"`

	// hostPort is optional and defaults to 0. 0 retains the traditional meaning in Marathon, which is "a random port from the range included in the Mesos resource offer". The resulting host ports for each task are exposed via the task details in the REST API and the Marathon web UI.
	HostPort int32 `json:"hostPort" protobuf:"varint,2,opt,name=hostPort"`

	//  is a helper port intended for doing service discovery using a well-known port per service. The assigned servicePort value is not used/interpreted by Marathon itself but supposed to used by load balancer infrastructure.
	ServicePort int32 `json:"servicePort,omitempty" protobuf:"varint,3,opt,name=servicePort"`

	// The "protocol" parameter is optional and defaults to "tcp". Its possible values are "tcp" and "udp"
	Protocol string `json:"protocol,omitempty" protobuf:"bytes,4,opt,name=protocol"`
}

// Parameters is the parameters to pass to the docker client when creating the container
type MarathonParameters struct {
	// Paramenter key
	Key string `json:"key,omitempty" protobuf:"bytes,1,opt,name=key"`

	// Paramenter value
	Value string `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`
}

// The upgradeStrategy controls how Marathon stops old versions and launches new versions.
type MarathonUpgradeStrategy struct {
	//  (Optional. Default: 1.0) - a number between 0and 1 that is multiplied with the instance count. This is the minimum number of healthy nodes that do not sacrifice overall application purpose. Marathon will make sure, during the upgrade process, that at any point of time this number of healthy instances are up.
	MinimumHealthCapacity float64 `json:"minimumHealthCapacity" protobuf:"bytes,1,opt,name=minimumHealthCapacity"`

	// (Optional. Default: 1.0) - a number between 0 and 1 which is multiplied with the instance count. This is the maximum number of additional instances launched at any point of time during the upgrade process.
	MaximumOverCapacity float64 `json:"maximumOverCapacity" protobuf:"bytes,2,opt,name=maximumOverCapacity"`
}

// Health checks to be performed by marathon on running tasks to determine if they are operating as expected.
type MarathonHealthCheck struct {
	// Command to run in order to determine the health of a task.
	Command *string `json:"command,omitempty" protobuf:"bytes,1,opt,name=command"`

	// (Optional. Default: 0): Index in this app's ports array to be used for health requests.
	PortIndex *int32 `json:"portIndex,omitempty" protobuf:"varint,2,opt,name=portIndex"`

	// (Optional. Default: "/"): Path to endpoint exposed by the task that will provide health status.
	Path *string `json:"path,omitempty" protobuf:"bytes,3,opt,name=path"`

	// (Optional. Default: 3) : Number of consecutive health check failures after which the unhealthy task should be killed.
	MaxConsecutiveFailures *int32 `json:"maxConsecutiveFailures,omitempty" protobuf:"varint,4,opt,name=maxConsecutiveFailures"`

	//  (Optional. Default: "HTTP"): Protocol of the requests to be performed. One of "HTTP", "HTTPS", "TCP", or "Command".
	Protocol string `json:"protocol,omitempty" protobuf:"bytes,5,opt,name=protocol"`

	// (Optional. Default: 15): Health check failures are ignored within this number of seconds of the task being started or until the task becomes healthy for the first time.
	GracePeriodSeconds int32 `json:"gracePeriodSeconds,omitempty" protobuf:"varint,6,opt,name=gracePeriodSeconds"`

	// (Optional. Default: 10): Number of seconds to wait between health checks.
	IntervalSeconds int32 `json:"intervalSeconds,omitempty" protobuf:"varint,7,opt,name=intervalSeconds"`

	// (Optional. Default: 20): Number of seconds after which a health check is considered a failure regardless of the response.
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty" protobuf:"varint,8,opt,name=timeoutSeconds"`
}

// Fetch will download URI before task starts
type MarathonFetch struct {
	// URI to be fetched by Mesos fetcher module
	URI string `json:"uri" protobuf:"bytes,1,opt,name=uri"`

	// Set fetched artifact as executable
	Executable bool `json:"executable" protobuf:"varint,2,opt,name=Executable"`

	// Extract fetched artifact if supported by Mesos fetcher mod
	Extract bool `json:"extract" protobuf:"varint,3,opt,name=extract"`

	// Cache fetched artifact if supported by Mesos fetcher module
	Cache bool `json:"cache" protobuf:"varint,4,opt,name=cache"`
}
