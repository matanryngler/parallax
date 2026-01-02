package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JobTemplateSpec struct {
	Image              string                        `json:"image"`
	Command            []string                      `json:"command"`
	EnvName            string                        `json:"envName"`
	Env                []corev1.EnvVar               `json:"env,omitempty"`
	EnvFrom            []corev1.EnvFromSource        `json:"envFrom,omitempty"`
	Resources          corev1.ResourceRequirements   `json:"resources,omitempty"`
	ServiceAccountName string                        `json:"serviceAccountName,omitempty"`
	ImagePullPolicy    corev1.PullPolicy             `json:"imagePullPolicy,omitempty"`
	ImagePullSecrets   []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	Tolerations        []corev1.Toleration           `json:"tolerations,omitempty"`
	Affinity           *corev1.Affinity              `json:"affinity,omitempty"`
	Labels             map[string]string             `json:"labels,omitempty"`
	Volumes            []corev1.Volume               `json:"volumes,omitempty"`
	VolumeMounts       []corev1.VolumeMount          `json:"volumeMounts,omitempty"`
	Ports              []corev1.ContainerPort        `json:"ports,omitempty"`
	InitContainers     []corev1.Container            `json:"initContainers,omitempty"`
	// InitImage is the container image to use for the internal parallax-init container (default: busybox:1.36)
	// +optional
	InitImage string `json:"initImage,omitempty"`
	// InitResources are the resource requirements for the internal parallax-init container
	// +optional
	InitResources corev1.ResourceRequirements `json:"initResources,omitempty"`
}

type ListJobSpec struct {
	ListSourceRef           string           `json:"listSourceRef,omitempty"`
	StaticList              []string         `json:"staticList,omitempty"`
	Parallelism             int32            `json:"parallelism"`
	Template                JobTemplateSpec  `json:"template"`
	TTLSecondsAfterFinished *int32           `json:"ttlSecondsAfterFinished,omitempty"`
	DeleteAfter             *metav1.Duration `json:"deleteAfter,omitempty"`
	BackoffLimit            *int32           `json:"backoffLimit,omitempty"`
	ActiveDeadlineSeconds   *int64           `json:"activeDeadlineSeconds,omitempty"`
}

type ListJobStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
	JobName    string             `json:"jobName,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type ListJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ListJobSpec   `json:"spec,omitempty"`
	Status ListJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ListJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ListJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ListJob{}, &ListJobList{})
}
