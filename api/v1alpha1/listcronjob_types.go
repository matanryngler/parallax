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
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ListCronJobSpec defines the desired state of ListCronJob.
type ListCronJobSpec struct {
	ListSourceRef              string                    `json:"listSourceRef,omitempty"`
	StaticList                 []string                  `json:"staticList,omitempty"`
	Parallelism                int32                     `json:"parallelism"`
	Template                   JobTemplateSpec           `json:"template"`
	TTLSecondsAfterFinished    *int32                    `json:"ttlSecondsAfterFinished,omitempty"`
	Schedule                   string                    `json:"schedule"`
	ConcurrencyPolicy          batchv1.ConcurrencyPolicy `json:"concurrencyPolicy,omitempty"`
	StartingDeadlineSeconds    *int64                    `json:"startingDeadlineSeconds,omitempty"`
	SuccessfulJobsHistoryLimit *int32                    `json:"successfulJobsHistoryLimit,omitempty"`
	FailedJobsHistoryLimit     *int32                    `json:"failedJobsHistoryLimit,omitempty"`
	Suspend                    *bool                     `json:"suspend,omitempty"`
}

// ListCronJobStatus defines the observed state of ListCronJob.
type ListCronJobStatus struct {
	Active           []corev1.ObjectReference `json:"active,omitempty"`
	LastScheduleTime *metav1.Time             `json:"lastScheduleTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ListCronJob is the Schema for the listcronjobs API.
type ListCronJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ListCronJobSpec   `json:"spec,omitempty"`
	Status ListCronJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ListCronJobList contains a list of ListCronJob.
type ListCronJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ListCronJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ListCronJob{}, &ListCronJobList{})
}
