package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JobTemplateSpec defines the template for Job pods that process list items.
// This simplified template is converted to a full Kubernetes Job specification by the controller.
type JobTemplateSpec struct {
	// Image is the container image to run for processing each item.
	// Should include the registry and tag for reproducibility.
	// Example: "myregistry/processor:v1.2.3"
	// Required field.
	Image string `json:"image"`

	// Command is the command to execute in the container.
	// Overrides the image's default ENTRYPOINT.
	// Example: ["/bin/sh", "-c", "process-item.sh"]
	// +optional
	Command []string `json:"command"`

	// EnvName is the name of the environment variable that will receive the item value.
	// The controller injects this environment variable with the item to process.
	// Default: "ITEM" if not specified.
	// Example: "CUSTOMER_ID", "FILE_PATH", "ORDER_ID"
	EnvName string `json:"envName"`

	// Env is a list of additional environment variables for the container.
	// These are added alongside the item environment variable.
	// Standard Kubernetes EnvVar format with support for valueFrom.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// EnvFrom sources to populate environment variables in the container.
	// Allows loading all keys from ConfigMaps or Secrets as environment variables.
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// Resources defines CPU and memory resource requests and limits.
	// Important: Always set requests and limits in production to ensure proper scheduling.
	// Example: requests: {cpu: "500m", memory: "256Mi"}, limits: {cpu: "2", memory: "512Mi"}
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// SecurityContext defines the security settings for the pod.
	// Includes runAsUser, runAsGroup, fsGroup, etc.
	// +optional
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`

	// ImagePullPolicy specifies when to pull the container image.
	// Valid values: Always, Never, IfNotPresent.
	// Default: IfNotPresent for tagged images, Always for :latest.
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets are references to secrets for pulling images from private registries.
	// Required for private Docker registries.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Tolerations allow pods to schedule onto nodes with matching taints.
	// Use for dedicated node pools, spot instances, or special hardware.
	// Example: Tolerating spot instance taints for cost savings.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Affinity defines pod scheduling constraints (node affinity, pod affinity/anti-affinity).
	// Use to control pod placement across nodes or in relation to other pods.
	// Example: Spreading jobs across availability zones.
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Labels are additional labels to apply to Job pods.
	// Useful for monitoring, cost tracking, and service mesh integration.
	// These are merged with controller-managed labels.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Volumes are volumes to attach to the pod.
	// Common uses: ConfigMaps, Secrets, PersistentVolumes, emptyDir.
	// Each volume must be mounted by VolumeMounts to be accessible.
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// VolumeMounts are mount points for volumes in the container.
	// Must reference volumes defined in the Volumes field.
	// Example: {name: "data", mountPath: "/data"}
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`

	// Ports are network ports to expose from the container.
	// Typically not needed for batch jobs, but useful for debugging or metrics.
	// +optional
	Ports []corev1.ContainerPort `json:"ports,omitempty"`

	// InitContainers are containers that run before the main container.
	// Common uses: setup tasks, waiting for dependencies, downloading data.
	// Init containers run sequentially and must complete successfully.
	// +optional
	InitContainers []corev1.Container `json:"initContainers,omitempty"`

	// InitImage is the container image to use for the internal parallax-init container.
	// This init container prepares the item environment variable.
	// Default: "busybox:1.36" if not specified.
	// Only override if you need a specific init container image.
	// +optional
	InitImage string `json:"initImage,omitempty"`

	// InitResources are the resource requirements for the internal parallax-init container.
	// Default: minimal resources (10m CPU, 10Mi memory) if not specified.
	// Increase if the init container performs complex setup.
	// +optional
	InitResources corev1.ResourceRequirements `json:"initResources,omitempty"`
}

// ListJobSpec defines the desired state of a ListJob.
// Specifies how to process items from a ListSource or static list in parallel.
type ListJobSpec struct {
	// ListSourceRef is the name of the ListSource to get items from.
	// The ListSource must exist in the same namespace.
	// Mutually exclusive with StaticList - use one or the other.
	// Example: "my-items" to reference a ListSource named "my-items"
	// +optional
	ListSourceRef string `json:"listSourceRef,omitempty"`

	// StaticList is a hardcoded list of items to process.
	// Mutually exclusive with ListSourceRef - use one or the other.
	// Useful for one-off jobs or testing without creating a ListSource.
	// Example: ["item1", "item2", "item3"]
	// +optional
	StaticList []string `json:"staticList,omitempty"`

	// Parallelism specifies the maximum number of jobs to run concurrently.
	// Controls resource consumption and processing speed.
	// Examples:
	//   - 1: Sequential processing (one at a time)
	//   - 5: Moderate parallelism (good for most workloads)
	//   - 20: High parallelism (for large clusters or quick turnaround)
	// Required field. Minimum value: 1.
	// +kubebuilder:validation:Minimum=1
	Parallelism int32 `json:"parallelism"`

	// Template defines the Job pod specification for processing items.
	// Each item creates a separate Job using this template.
	// The item value is injected as an environment variable.
	// Required field.
	Template JobTemplateSpec `json:"template"`

	// TTLSecondsAfterFinished is the time in seconds to keep completed/failed Jobs.
	// After this duration, the Job and its pods are automatically deleted.
	// Helps prevent cluster clutter from old Jobs.
	// Examples:
	//   - 300: 5 minutes (quick cleanup for dev)
	//   - 3600: 1 hour (standard for production)
	//   - 86400: 24 hours (for debugging or auditing)
	// If not specified, Jobs are not automatically deleted.
	// +optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`

	// DeleteAfter is an alternative to TTLSecondsAfterFinished using Duration format.
	// Example: "30m" for 30 minutes, "2h" for 2 hours.
	// If both are specified, TTLSecondsAfterFinished takes precedence.
	// +optional
	DeleteAfter *metav1.Duration `json:"deleteAfter,omitempty"`

	// BackoffLimit specifies the number of retries before marking a Job as failed.
	// Each retry creates a new pod after the previous one fails.
	// Default: 6 (Kubernetes default) if not specified.
	// Set to 0 for no retries (fail immediately).
	// Set higher for transient failures (network issues, rate limits).
	// +optional
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// ActiveDeadlineSeconds is the maximum duration (in seconds) for the Job.
	// The Job is terminated if it runs longer than this limit.
	// Helps prevent runaway or stuck jobs.
	// Examples:
	//   - 600: 10 minutes
	//   - 3600: 1 hour
	//   - 7200: 2 hours
	// If not specified, Jobs run indefinitely until completion or failure.
	// +optional
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty"`
}

// ListJobStatus defines the observed state of a ListJob.
// Updated by the controller to reflect Job creation and execution status.
type ListJobStatus struct {
	// Conditions represent the latest available observations of the ListJob's state.
	// Standard condition types:
	//   - Complete: True if all Jobs have completed successfully
	//   - Failed: True if any Jobs failed after retries
	//   - Running: True if Jobs are currently executing
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// JobName is the name of the Kubernetes Job created for this ListJob.
	// Use to query Job status: kubectl get job <jobName>
	// +optional
	JobName string `json:"jobName,omitempty"`
}

// ListJob is a Kubernetes custom resource that processes items from a ListSource in parallel.
// Creates Kubernetes Jobs with indexed completion mode, where each job processes one item.
//
// The controller:
//  1. Fetches items from the referenced ListSource (or uses StaticList)
//  2. Creates a Kubernetes Job with indexed completion mode
//  3. Each job index corresponds to one item
//  4. Items are injected as environment variables in job pods
//  5. Jobs run in parallel up to the specified Parallelism limit
//
// Example usage:
//
//	apiVersion: batchops.example.com/v1alpha1
//	kind: ListJob
//	metadata:
//	  name: process-orders
//	spec:
//	  listSourceRef: order-list
//	  parallelism: 5
//	  template:
//	    image: myapp/processor:v1
//	    command: ["process-order.sh"]
//	    envName: ORDER_ID
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type ListJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the ListJob.
	// Specifies items to process and job configuration.
	Spec ListJobSpec `json:"spec,omitempty"`

	// Status represents the observed state of the ListJob.
	// Updated by the controller with Job creation status.
	Status ListJobStatus `json:"status,omitempty"`
}

// ListJobList contains a list of ListJob resources.
// Used by the Kubernetes API for list operations.
// +kubebuilder:object:root=true
type ListJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of ListJob resources.
	Items []ListJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ListJob{}, &ListJobList{})
}
