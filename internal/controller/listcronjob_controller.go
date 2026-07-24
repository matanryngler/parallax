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

package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	batchopsv1alpha1 "github.com/matanryngler/parallax/api/v1alpha1"
	"github.com/matanryngler/parallax/internal/metrics"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ListCronJobReconciler reconciles a ListCronJob object
type ListCronJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	listCronJobFinalizer = "listcronjob.batchops.io/finalizer"
	// defaultInitImage is the default busybox image used for the init container.
	// Uses busybox:1 to get latest 1.x patch versions while maintaining compatibility.
	// Users can override via spec.template.initImage in the CRD.
	defaultInitImage = "busybox:1"
)

// +kubebuilder:rbac:groups=batchops.io,resources=listcronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batchops.io,resources=listcronjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batchops.io,resources=listcronjobs/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ListCronJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *ListCronJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling ListCronJob",
		"name", req.Name,
		"namespace", req.Namespace,
	)

	var listCronJob batchopsv1alpha1.ListCronJob
	if err := r.Get(ctx, req.NamespacedName, &listCronJob); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("ListCronJob not found. Likely deleted.", "name", req.Name, "namespace", req.Namespace)
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get ListCronJob")
		return ctrl.Result{}, err
	}

	log.Info("Reconciling ListCronJob details",
		"generation", listCronJob.Generation,
		"resourceVersion", listCronJob.ResourceVersion,
		"deletionTimestamp", listCronJob.DeletionTimestamp,
		"finalizers", listCronJob.Finalizers,
	)

	if !listCronJob.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&listCronJob, listCronJobFinalizer) {
			log.Info("Cleaning up child resources before deletion")
			cronJob := &batchv1.CronJob{ObjectMeta: metav1.ObjectMeta{Name: listCronJob.Name, Namespace: listCronJob.Namespace}}
			_ = r.Delete(ctx, cronJob)
			cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-list", listCronJob.Name), Namespace: listCronJob.Namespace}}
			_ = r.Delete(ctx, cm)

			controllerutil.RemoveFinalizer(&listCronJob, listCronJobFinalizer)
			if err := r.Update(ctx, &listCronJob); err != nil {
				log.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(&listCronJob, listCronJobFinalizer) {
		controllerutil.AddFinalizer(&listCronJob, listCronJobFinalizer)
		if err := r.Update(ctx, &listCronJob); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	var list []string
	if len(listCronJob.Spec.StaticList) > 0 {
		list = listCronJob.Spec.StaticList
	} else if listCronJob.Spec.ListSourceRef != "" {
		// Get the ListSource's ConfigMap
		var listSourceCM corev1.ConfigMap
		if err := r.Get(ctx, client.ObjectKey{Name: listCronJob.Spec.ListSourceRef, Namespace: req.Namespace}, &listSourceCM); err != nil {
			log.Error(err, "Failed to get ListSource ConfigMap", "configMap", listCronJob.Spec.ListSourceRef)
			return ctrl.Result{}, err
		}

		// Parse the items from the ConfigMap
		itemsStr := listSourceCM.Data["items"]
		if itemsStr == "" {
			log.Error(nil, "ListSource ConfigMap has no items", "configMap", listCronJob.Spec.ListSourceRef)
			return ctrl.Result{}, fmt.Errorf("ListSource ConfigMap has no items")
		}

		// Split by newlines and trim whitespace
		list = strings.Split(itemsStr, "\n")
		for i, item := range list {
			list[i] = strings.TrimSpace(item)
		}
	} else {
		log.Error(nil, "Neither StaticList nor ListSourceRef specified")
		return ctrl.Result{}, fmt.Errorf("either StaticList or ListSourceRef must be specified")
	}

	// Create ConfigMap with newline-separated items
	jobCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-list", listCronJob.Name),
			Namespace: req.Namespace,
		},
		Data: map[string]string{
			"items": strings.Join(list, "\n"),
		},
	}
	if err := ctrl.SetControllerReference(&listCronJob, jobCm, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Create(ctx, jobCm); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			log.Error(err, "Failed to create ConfigMap")
			return ctrl.Result{}, err
		}
		// Update existing ConfigMap
		existingCm := &corev1.ConfigMap{}
		if err := r.Get(ctx, client.ObjectKey{Name: jobCm.Name, Namespace: jobCm.Namespace}, existingCm); err != nil {
			log.Error(err, "Failed to get existing ConfigMap")
			return ctrl.Result{}, err
		}
		existingCm.Data = jobCm.Data
		if err := r.Update(ctx, existingCm); err != nil {
			log.Error(err, "Failed to update ConfigMap")
			return ctrl.Result{}, err
		}
		log.Info("Updated ConfigMap with new items",
			"configmap", jobCm.Name,
			"items", strings.Join(list, ","),
			"itemCount", len(list),
			"resourceVersion", existingCm.ResourceVersion)
	} else {
		log.Info("Created new ConfigMap with items",
			"configmap", jobCm.Name,
			"items", strings.Join(list, ","),
			"itemCount", len(list),
			"resourceVersion", jobCm.ResourceVersion)
	}

	envName := listCronJob.Spec.Template.EnvName
	if envName == "" {
		envName = "ITEM"
	}

	// Prepare volumes - start with required volumes, then add user-specified ones
	volumes := []corev1.Volume{
		{
			Name: "list",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: jobCm.Name},
					Optional:             func() *bool { b := true; return &b }(),
				},
			},
		},
		{
			Name:         "shared",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		},
	}
	volumes = append(volumes, listCronJob.Spec.Template.Volumes...)

	// Prepare volume mounts for main container - start with required mount, then add user-specified ones
	mainVolumeMounts := []corev1.VolumeMount{
		{Name: "shared", MountPath: "/shared"},
	}
	mainVolumeMounts = append(mainVolumeMounts, listCronJob.Spec.Template.VolumeMounts...)

	// Prepare init containers - start with user-specified ones, then add the required internal init container last
	initContainers := make([]corev1.Container, len(listCronJob.Spec.Template.InitContainers))
	copy(initContainers, listCronJob.Spec.Template.InitContainers)

	// Determine init image (use default if not specified)
	initImage := listCronJob.Spec.Template.InitImage
	if initImage == "" {
		initImage = defaultInitImage
	}

	initContainers = append(initContainers, corev1.Container{
		Name:  "parallax-init",
		Image: initImage,
		Command: []string{"sh", "-c", fmt.Sprintf(`
			# Read the items file
			ITEMS=$(cat /list/items)
			# Get the item at the given index (0-based)
			VAL=$(echo "$ITEMS" | sed -n "$((JOB_COMPLETION_INDEX+1))p")
			# Export the value
			echo "export %s=$VAL" > /shared/env.sh
		`, envName)},
		Env: []corev1.EnvVar{
			{
				Name: "JOB_COMPLETION_INDEX",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.annotations['batch.kubernetes.io/job-completion-index']",
					},
				},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "list", MountPath: "/list", ReadOnly: true},
			{Name: "shared", MountPath: "/shared"},
		},
		Resources: listCronJob.Spec.Template.InitResources,
	})

	mainCommand := append([]string{"sh", "-c", ". /shared/env.sh && exec \"$@\"", "--"}, listCronJob.Spec.Template.Command...)

	podSpec := corev1.PodSpec{
		ServiceAccountName: listCronJob.Spec.Template.ServiceAccountName,
		SecurityContext:    listCronJob.Spec.Template.SecurityContext,
		ImagePullSecrets:   listCronJob.Spec.Template.ImagePullSecrets,
		Tolerations:        listCronJob.Spec.Template.Tolerations,
		Affinity:           listCronJob.Spec.Template.Affinity,
		Volumes:            volumes,
		InitContainers:     initContainers,
		Containers: []corev1.Container{
			{
				Name:            "main",
				Image:           listCronJob.Spec.Template.Image,
				ImagePullPolicy: listCronJob.Spec.Template.ImagePullPolicy,
				Command:         mainCommand,
				Resources:       listCronJob.Spec.Template.Resources,
				Env:             listCronJob.Spec.Template.Env,
				EnvFrom:         listCronJob.Spec.Template.EnvFrom,
				VolumeMounts:    mainVolumeMounts,
				Ports:           listCronJob.Spec.Template.Ports,
			},
		},
		RestartPolicy: corev1.RestartPolicyNever,
	}

	// Merge labels for the Job pods
	podLabels := make(map[string]string)
	for k, v := range listCronJob.Spec.Template.Labels {
		podLabels[k] = v
	}
	podLabels["listcronjob.batchops.io/name"] = listCronJob.Name

	jobSpec := batchv1.JobSpec{
		Parallelism:             &listCronJob.Spec.Parallelism,
		Completions:             &[]int32{int32(len(list))}[0],
		CompletionMode:          func() *batchv1.CompletionMode { mode := batchv1.IndexedCompletion; return &mode }(),
		TTLSecondsAfterFinished: listCronJob.Spec.TTLSecondsAfterFinished,
		BackoffLimit:            listCronJob.Spec.BackoffLimit,
		ActiveDeadlineSeconds:   listCronJob.Spec.ActiveDeadlineSeconds,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: podLabels,
			},
			Spec: podSpec,
		},
	}

	// Create or update CronJob
	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      listCronJob.Name,
			Namespace: req.Namespace,
			Labels: map[string]string{
				"listcronjob.batchops.io/name": listCronJob.Name,
			},
		},
		Spec: batchv1.CronJobSpec{
			Schedule:                   listCronJob.Spec.Schedule,
			ConcurrencyPolicy:          batchv1.ConcurrencyPolicy(listCronJob.Spec.ConcurrencyPolicy),
			StartingDeadlineSeconds:    listCronJob.Spec.StartingDeadlineSeconds,
			SuccessfulJobsHistoryLimit: listCronJob.Spec.SuccessfulJobsHistoryLimit,
			FailedJobsHistoryLimit:     listCronJob.Spec.FailedJobsHistoryLimit,
			Suspend:                    listCronJob.Spec.Suspend,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: jobSpec,
			},
		},
	}

	if err := ctrl.SetControllerReference(&listCronJob, cronJob, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// Check if CronJob already exists
	existingCronJob := &batchv1.CronJob{}
	err := r.Get(ctx, client.ObjectKey{Name: listCronJob.Name, Namespace: req.Namespace}, existingCronJob)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create new CronJob
			if err := r.Create(ctx, cronJob); err != nil {
				log.Error(err, "Failed to create CronJob")
				return ctrl.Result{}, err
			}
			log.Info("Created new CronJob",
				"name", cronJob.Name,
				"completions", *jobSpec.Completions,
				"configmap", jobCm.Name,
				"configmapVersion", jobCm.ResourceVersion)
		} else {
			log.Error(err, "Failed to get CronJob")
			return ctrl.Result{}, err
		}
	} else {
		// Update existing CronJob
		existingCronJob.Spec = cronJob.Spec
		// Force a new revision by updating the template
		existingCronJob.Spec.JobTemplate.Spec.Template.ObjectMeta.Annotations = map[string]string{
			"configmap-version": jobCm.ResourceVersion,
		}
		if err := r.Update(ctx, existingCronJob); err != nil {
			log.Error(err, "Failed to update CronJob")
			return ctrl.Result{}, err
		}
		log.Info("Updated existing CronJob",
			"name", cronJob.Name,
			"completions", *jobSpec.Completions,
			"configmap", jobCm.Name,
			"configmapVersion", jobCm.ResourceVersion,
			"oldCompletions", *existingCronJob.Spec.JobTemplate.Spec.Completions)
		cronJob = existingCronJob
	}

	// Track cycle starts and skips
	if err := r.trackCronJobCycles(ctx, &listCronJob, cronJob); err != nil {
		log.Error(err, "Failed to track CronJob cycles")
		// Don't fail reconciliation for tracking errors
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ListCronJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Setup field indexer for Events by involvedObject.name
	// This allows efficient querying of events for a specific CronJob
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Event{}, "involvedObject.name", func(obj client.Object) []string {
		event := obj.(*corev1.Event)
		return []string{event.InvolvedObject.Name}
	}); err != nil {
		return fmt.Errorf("failed to setup field indexer for events: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&batchopsv1alpha1.ListCronJob{}).
		Watches(
			&corev1.ConfigMap{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForConfigMap),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&batchv1.CronJob{},
			handler.EnqueueRequestsFromMapFunc(r.findListCronJobForCronJob),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

// trackCronJobCycles detects and logs new cycle starts and skipped cycles
func (r *ListCronJobReconciler) trackCronJobCycles(ctx context.Context, listCronJob *batchopsv1alpha1.ListCronJob, cronJob *batchv1.CronJob) error {
	log := ctrl.LoggerFrom(ctx)

	// Track active pods metric - count active pods across all active jobs
	var totalActivePods int32
	for _, jobRef := range cronJob.Status.Active {
		job := &batchv1.Job{}
		err := r.Get(ctx, types.NamespacedName{Name: jobRef.Name, Namespace: jobRef.Namespace}, job)
		if err != nil {
			log.V(1).Info("Failed to get active job for metrics", "job", jobRef.Name, "error", err)
			continue
		}
		totalActivePods += job.Status.Active
	}
	metrics.CronJobActivePods.WithLabelValues(listCronJob.Name, listCronJob.Namespace).Set(float64(totalActivePods))

	// Track cycle duration when cycle completes (active jobs = 0)
	if len(cronJob.Status.Active) == 0 && listCronJob.Status.LastScheduleTime != nil {
		// Calculate actual execution time (from schedule start to completion)
		duration := time.Since(listCronJob.Status.LastScheduleTime.Time).Seconds()
		metrics.CronJobCycleDuration.WithLabelValues(
			listCronJob.Name,
			listCronJob.Namespace,
		).Set(duration)
	}

	// Check if a new cycle started
	if cronJob.Status.LastScheduleTime != nil {
		// Compare with our tracked last schedule time
		if listCronJob.Status.LastScheduleTime == nil ||
			!cronJob.Status.LastScheduleTime.Equal(listCronJob.Status.LastScheduleTime) {

			// New cycle started!
			log.Info("New CronJob cycle started",
				"listcronjob", listCronJob.Name,
				"cronjob", cronJob.Name,
				"schedule", listCronJob.Spec.Schedule,
				"scheduleTime", cronJob.Status.LastScheduleTime.Time,
				"activeJobs", len(cronJob.Status.Active))

			// Increment cycle started metric
			metrics.CronJobCyclesStarted.WithLabelValues(
				listCronJob.Name,
				listCronJob.Namespace,
			).Inc()

			// Update our tracked time with retry on conflict
			newScheduleTime := cronJob.Status.LastScheduleTime
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := r.Get(ctx, client.ObjectKeyFromObject(listCronJob), listCronJob); err != nil {
					return err
				}
				listCronJob.Status.LastScheduleTime = newScheduleTime
				return r.Status().Update(ctx, listCronJob)
			})
			if err != nil {
				return fmt.Errorf("failed to update LastScheduleTime: %w", err)
			}
		}
	}

	// Detect skipped cycles for Forbid policy by querying Events for this specific CronJob
	if listCronJob.Spec.ConcurrencyPolicy == batchv1.ForbidConcurrent {
		if err := r.detectSkippedCycles(ctx, listCronJob, cronJob); err != nil {
			log.V(1).Info("Failed to detect skipped cycles", "error", err)
			// Don't fail reconciliation for this
		}
	}

	return nil
}

// detectSkippedCycles queries Events for the specific CronJob and logs any JobAlreadyActive events
func (r *ListCronJobReconciler) detectSkippedCycles(ctx context.Context, listCronJob *batchopsv1alpha1.ListCronJob, cronJob *batchv1.CronJob) error {
	log := ctrl.LoggerFrom(ctx)

	// List events for this specific CronJob using field selector
	eventList := &corev1.EventList{}
	if err := r.List(ctx, eventList,
		client.InNamespace(cronJob.Namespace),
		client.MatchingFields{"involvedObject.name": cronJob.Name},
	); err != nil {
		return fmt.Errorf("failed to list events for CronJob: %w", err)
	}

	// Look for recent JobAlreadyActive events
	now := time.Now()
	recentWindow := 5 * time.Minute // Only consider events from last 5 minutes

	for _, event := range eventList.Items {
		// Filter for JobAlreadyActive events on CronJobs
		if event.Reason != "JobAlreadyActive" ||
			event.InvolvedObject.Kind != "CronJob" ||
			event.InvolvedObject.APIVersion != "batch/v1" {
			continue
		}

		// Check if event is recent
		eventTime := event.LastTimestamp.Time
		if eventTime.IsZero() {
			eventTime = event.EventTime.Time
		}
		if now.Sub(eventTime) > recentWindow {
			continue
		}

		// Check if we've already processed this event
		eventUID := string(event.UID)
		if listCronJob.Status.LastSkipEventUID == eventUID {
			continue
		}

		// New skip event found - log it!
		log.Info("CronJob schedule skipped due to active jobs (Forbid policy)",
			"listcronjob", listCronJob.Name,
			"cronjob", cronJob.Name,
			"namespace", cronJob.Namespace,
			"schedule", listCronJob.Spec.Schedule,
			"eventTime", eventTime,
			"eventMessage", event.Message)

		// Increment skip metric
		metrics.CronJobCyclesSkipped.WithLabelValues(
			listCronJob.Name,
			listCronJob.Namespace,
			"forbid_policy",
		).Inc()

		// Update status to track this event (with retry on conflict)
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := r.Get(ctx, client.ObjectKeyFromObject(listCronJob), listCronJob); err != nil {
				return err
			}
			listCronJob.Status.LastSkipEventUID = eventUID
			return r.Status().Update(ctx, listCronJob)
		})
		if err != nil {
			return fmt.Errorf("failed to update LastSkipEventUID: %w", err)
		}

		// Only process the most recent event
		break
	}

	return nil
}

// findObjectsForConfigMap maps a ConfigMap to ListCronJobs that reference it
func (r *ListCronJobReconciler) findObjectsForConfigMap(ctx context.Context, obj client.Object) []reconcile.Request {
	configMap, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return []reconcile.Request{}
	}

	// Get all ListCronJobs in the same namespace
	var listCronJobs batchopsv1alpha1.ListCronJobList
	if err := r.List(ctx, &listCronJobs, client.InNamespace(configMap.Namespace)); err != nil {
		return []reconcile.Request{}
	}

	requests := []reconcile.Request{}
	for _, listCronJob := range listCronJobs.Items {
		// Check if this ListCronJob references our ConfigMap
		if listCronJob.Spec.ListSourceRef == configMap.Name {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      listCronJob.Name,
					Namespace: listCronJob.Namespace,
				},
			})
		}
	}

	return requests
}

// findListCronJobForCronJob maps a CronJob to its owning ListCronJob
func (r *ListCronJobReconciler) findListCronJobForCronJob(ctx context.Context, obj client.Object) []reconcile.Request {
	cronJob, ok := obj.(*batchv1.CronJob)
	if !ok {
		return []reconcile.Request{}
	}

	// Find the owning ListCronJob via OwnerReferences
	for _, owner := range cronJob.OwnerReferences {
		if owner.Kind == "ListCronJob" && owner.APIVersion == "batchops.io/v1alpha1" {
			return []reconcile.Request{{
				NamespacedName: types.NamespacedName{
					Name:      owner.Name,
					Namespace: cronJob.Namespace,
				},
			}}
		}
	}

	return []reconcile.Request{}
}
