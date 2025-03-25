/*
Copyright 2025.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JobTemplateSpec struct {
	Image   string   `json:"image"`
	Command []string `json:"command"`
	EnvName string   `json:"envName"`
}

type ListJobSpec struct {
	ListSourceRef string          `json:"listSourceRef,omitempty"`
	StaticList    []string        `json:"staticList,omitempty"`
	Parallelism   int32           `json:"parallelism"`
	Template      JobTemplateSpec `json:"template"`
}

type ListJobStatus struct {
	JobName string `json:"jobName,omitempty"`
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
