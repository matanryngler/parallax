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

	batchopsv1alpha1 "github.com/matanryngler/parallax/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

const listCronJobFinalizer = "listcronjob.batchops.io/finalizer"

// +kubebuilder:rbac:groups=batchops.io,resources=listcronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batchops.io,resources=listcronjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batchops.io,resources=listcronjobs/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps/status,verbs=get;list;watch

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

	podSpec := corev1.PodSpec{
		Volumes: []corev1.Volume{
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
		},
		InitContainers: []corev1.Container{
			{
				Name:  "init",
				Image: "busybox",
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
			},
		},
		Containers: []corev1.Container{
			{
				Name:      "main",
				Image:     listCronJob.Spec.Template.Image,
				Command:   []string{"sh", "-c", ". /shared/env.sh && " + strings.Join(listCronJob.Spec.Template.Command, " ")},
				Resources: listCronJob.Spec.Template.Resources,
				VolumeMounts: []corev1.VolumeMount{
					{Name: "shared", MountPath: "/shared"},
				},
			},
		},
		RestartPolicy: corev1.RestartPolicyNever,
	}

	jobSpec := batchv1.JobSpec{
		Parallelism:             &listCronJob.Spec.Parallelism,
		Completions:             &[]int32{int32(len(list))}[0],
		CompletionMode:          func() *batchv1.CompletionMode { mode := batchv1.IndexedCompletion; return &mode }(),
		TTLSecondsAfterFinished: listCronJob.Spec.TTLSecondsAfterFinished,
		Template: corev1.PodTemplateSpec{
			Spec: podSpec,
		},
	}

	// Create or update CronJob
	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      listCronJob.Name,
			Namespace: req.Namespace,
			Labels: map[string]string{
				"listcronjob": listCronJob.Name,
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
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ListCronJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchopsv1alpha1.ListCronJob{}).
		Watches(
			&corev1.ConfigMap{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForConfigMap),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
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
