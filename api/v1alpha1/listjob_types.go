package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JobTemplateSpec struct {
	Image     string                      `json:"image"`
	Command   []string                    `json:"command"`
	EnvName   string                      `json:"envName"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type ListJobSpec struct {
	ListSourceRef           string           `json:"listSourceRef,omitempty"`
	StaticList              []string         `json:"staticList,omitempty"`
	Parallelism             int32            `json:"parallelism"`
	Template                JobTemplateSpec  `json:"template"`
	TTLSecondsAfterFinished *int32           `json:"ttlSecondsAfterFinished,omitempty"`
	DeleteAfter             *metav1.Duration `json:"deleteAfter,omitempty"`
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
