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

// ListCronJobSpec defines the desired state of ListCronJob.
// Combines scheduling configuration with ListJob processing to run batch jobs on a schedule.
type ListCronJobSpec struct {
	// ListSourceRef is the name of the ListSource to get items from.
	// The ListSource must exist in the same namespace.
	// Mutually exclusive with StaticList - use one or the other.
	// The data from the ListSource is fetched fresh on each scheduled run.
	// +optional
	ListSourceRef string `json:"listSourceRef,omitempty"`

	// StaticList is a hardcoded list of items to process.
	// Mutually exclusive with ListSourceRef - use one or the other.
	// The same static list is used for every scheduled run.
	// +optional
	StaticList []string `json:"staticList,omitempty"`

	// Parallelism specifies the maximum number of jobs to run concurrently per schedule.
	// Controls resource consumption for each batch run.
	// Example: 5 means up to 5 items are processed in parallel during each run.
	// Required field. Minimum value: 1.
	// +kubebuilder:validation:Minimum=1
	Parallelism int32 `json:"parallelism"`

	// Template defines the Job pod specification for processing items.
	// Each scheduled run creates Jobs using this template.
	// Required field.
	Template JobTemplateSpec `json:"template"`

	// TTLSecondsAfterFinished is the time in seconds to keep completed/failed Jobs.
	// After this duration, Jobs and their pods are automatically deleted.
	// Applies to Jobs created by each scheduled run.
	// Example: 3600 for 1 hour, 86400 for 24 hours.
	// +optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`

	// Schedule defines when to run the job using cron syntax.
	// Format: "minute hour day month dayOfWeek" (standard cron format).
	// Examples:
	//   - "0 * * * *": Every hour
	//   - "*/5 * * * *": Every 5 minutes
	//   - "0 0 * * *": Daily at midnight UTC
	//   - "0 2 * * 0": Weekly on Sunday at 2 AM UTC
	//   - "0 0 1 * *": Monthly on the 1st at midnight UTC
	// Required field. Use crontab.guru to validate expressions.
	// +kubebuilder:validation:Required
	Schedule string `json:"schedule"`

	// ConcurrencyPolicy specifies how to handle concurrent executions.
	// Valid values:
	//   - Allow: Multiple ListJobs can run simultaneously (default if not specified)
	//   - Forbid: Skip new run if previous run is still active
	//   - Replace: Cancel current run and start new one immediately
	// Default: Forbid (safer for most use cases to prevent overlapping runs).
	// +kubebuilder:default=Forbid
	// +optional
	ConcurrencyPolicy batchv1.ConcurrencyPolicy `json:"concurrencyPolicy,omitempty"`

	// StartingDeadlineSeconds is the deadline in seconds for starting a missed job.
	// If a job misses its scheduled time (e.g., scheduler was down), it will only start
	// if it can begin within this deadline. Missed jobs beyond this deadline are skipped.
	// Example: 300 means a job must start within 5 minutes of its scheduled time.
	// If not specified, missed jobs can start at any time.
	// +optional
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty"`

	// SuccessfulJobsHistoryLimit is the number of successful ListJobs to retain.
	// Older successful runs are automatically deleted to prevent clutter.
	// Default: 3 if not specified.
	// Set to 0 to delete successful jobs immediately.
	// Set higher (e.g., 10) for debugging or audit trails.
	// +optional
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty"`

	// FailedJobsHistoryLimit is the number of failed ListJobs to retain.
	// Older failed runs are automatically deleted.
	// Default: 1 if not specified.
	// Increase for troubleshooting repeated failures.
	// +optional
	FailedJobsHistoryLimit *int32 `json:"failedJobsHistoryLimit,omitempty"`

	// Suspend tells the controller to suspend subsequent runs.
	// Existing active runs are not affected.
	// Useful for maintenance windows or temporarily disabling scheduled jobs.
	// Set to true to pause scheduling, false to resume.
	// Default: false (scheduling active).
	// +optional
	Suspend *bool `json:"suspend,omitempty"`

	// BackoffLimit specifies the number of retries before marking a Job as failed.
	// Applies to Jobs created by each scheduled run.
	// Default: 6 (Kubernetes default) if not specified.
	// +optional
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// ActiveDeadlineSeconds is the maximum duration (in seconds) for each Job.
	// Jobs are terminated if they run longer than this limit.
	// Applies to Jobs created by each scheduled run.
	// Example: 3600 for 1 hour maximum per run.
	// +optional
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty"`
}

// ListCronJobStatus defines the observed state of ListCronJob.
// Updated by the controller to reflect scheduling and execution status.
type ListCronJobStatus struct {
	// Conditions represent the latest available observations of the ListCronJob's state.
	// Standard condition types:
	//   - Scheduled: True if the CronJob is being scheduled normally
	//   - Suspended: True if the CronJob is suspended
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// Active lists currently running ListJobs created by this CronJob.
	// Each reference points to a ListJob that hasn't completed yet.
	// Empty if no jobs are currently running.
	// Use to monitor concurrent executions and enforce ConcurrencyPolicy.
	// +optional
	Active []corev1.ObjectReference `json:"active,omitempty"`

	// LastScheduleTime is the timestamp of the last time a ListJob was created.
	// Updated each time the schedule triggers and a new run is created.
	// Use to verify the CronJob is firing on schedule.
	// If this timestamp is old, check if the CronJob is suspended or has errors.
	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`

	// LastSkipEventUID tracks the UID of the last JobAlreadyActive event processed.
	// Used internally by the controller to avoid duplicate logging of skip events.
	// When ConcurrencyPolicy is Forbid and a job is already active, the controller
	// logs a skip event once per occurrence.
	// +optional
	LastSkipEventUID string `json:"lastSkipEventUID,omitempty"`
}

// ListCronJob is a Kubernetes custom resource that schedules ListJobs to run on a cron schedule.
// Combines standard cron scheduling with parallel batch processing from ListJobs.
//
// The controller:
//  1. Monitors the schedule and triggers at specified times
//  2. Creates a ListJob when the schedule fires
//  3. The ListJob fetches fresh data from the ListSource
//  4. Jobs run in parallel to process items
//  5. History is maintained according to HistoryLimit settings
//
// Similar to Kubernetes CronJob but creates ListJobs instead of regular Jobs.
//
// Example usage:
//
//	apiVersion: batchops.example.com/v1alpha1
//	kind: ListCronJob
//	metadata:
//	  name: daily-report
//	spec:
//	  schedule: "0 0 * * *"  # Daily at midnight
//	  concurrencyPolicy: Forbid
//	  listSourceRef: report-data
//	  parallelism: 5
//	  template:
//	    image: myapp/reporter:v1
//	    command: ["generate-report.sh"]
//	    envName: REPORT_ID
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type ListCronJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the ListCronJob.
	// Specifies scheduling and job configuration.
	Spec ListCronJobSpec `json:"spec,omitempty"`

	// Status represents the observed state of the ListCronJob.
	// Updated by the controller with scheduling information.
	Status ListCronJobStatus `json:"status,omitempty"`
}

// ListCronJobList contains a list of ListCronJob resources.
// Used by the Kubernetes API for list operations.
// +kubebuilder:object:root=true
type ListCronJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of ListCronJob resources.
	Items []ListCronJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ListCronJob{}, &ListCronJobList{})
}
