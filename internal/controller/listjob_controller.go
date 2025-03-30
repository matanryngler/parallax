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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ListJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const listJobFinalizer = "listjob.batchops.io/finalizer"

func (r *ListJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling ListJob",
		"name", req.Name,
		"namespace", req.Namespace,
	)

	var listJob batchopsv1alpha1.ListJob
	if err := r.Get(ctx, req.NamespacedName, &listJob); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("ListJob not found. Likely deleted.", "name", req.Name, "namespace", req.Namespace)
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get ListJob")
		return ctrl.Result{}, err
	}

	log.Info("Reconciling ListJob details",
		"generation", listJob.Generation,
		"resourceVersion", listJob.ResourceVersion,
		"deletionTimestamp", listJob.DeletionTimestamp,
		"finalizers", listJob.Finalizers,
	)

	if !listJob.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&listJob, listJobFinalizer) {
			log.Info("Cleaning up child resources before deletion")
			job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: listJob.Name, Namespace: listJob.Namespace}}
			_ = r.Delete(ctx, job, &client.DeleteOptions{
				PropagationPolicy: func() *metav1.DeletionPropagation {
					policy := metav1.DeletePropagationBackground
					return &policy
				}(),
			})
			cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-list", listJob.Name), Namespace: listJob.Namespace}}
			_ = r.Delete(ctx, cm)

			controllerutil.RemoveFinalizer(&listJob, listJobFinalizer)
			if err := r.Update(ctx, &listJob); err != nil {
				log.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(&listJob, listJobFinalizer) {
		controllerutil.AddFinalizer(&listJob, listJobFinalizer)
		if err := r.Update(ctx, &listJob); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if listJob.Spec.DeleteAfter != nil {
		expiry := listJob.CreationTimestamp.Add(listJob.Spec.DeleteAfter.Duration)
		if metav1.Now().After(expiry) {
			log.Info("Deleting ListJob due to DeleteAfter expiry")
			if err := r.Delete(ctx, &listJob); err != nil {
				log.Error(err, "Failed to delete expired ListJob")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}

	var list []string
	if len(listJob.Spec.StaticList) > 0 {
		list = listJob.Spec.StaticList
	} else {
		var cm corev1.ConfigMap
		err := r.Get(ctx, client.ObjectKey{Name: listJob.Spec.ListSourceRef, Namespace: req.Namespace}, &cm)
		if err != nil {
			log.Error(err, "Failed to fetch ListSource ConfigMap")
			return ctrl.Result{}, err
		}
		listStr := cm.Data["items"]
		list = strings.Split(listStr, ",")
	}

	jobCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-list", listJob.Name),
			Namespace: req.Namespace,
		},
		Data: map[string]string{
			"items.txt": strings.Join(list, ","),
		},
	}
	if err := ctrl.SetControllerReference(&listJob, jobCm, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Create(ctx, jobCm); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			log.Error(err, "Failed to create ConfigMap")
			return ctrl.Result{}, err
		}
	}

	envName := listJob.Spec.Template.EnvName
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
				Name:    "init",
				Image:   "busybox",
				Command: []string{"sh", "-c", fmt.Sprintf("VAL=$(cut -d',' -f$((`echo $JOB_COMPLETION_INDEX`+1)) /list/items.txt); echo \"export %s=$VAL\" > /shared/env.sh", envName)},
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
					{Name: "list", MountPath: "/list"},
					{Name: "shared", MountPath: "/shared"},
				},
			},
		},
		Containers: []corev1.Container{
			{
				Name:      "main",
				Image:     listJob.Spec.Template.Image,
				Command:   []string{"sh", "-c", ". /shared/env.sh && " + strings.Join(listJob.Spec.Template.Command, " ")},
				Resources: listJob.Spec.Template.Resources,
				VolumeMounts: []corev1.VolumeMount{
					{Name: "shared", MountPath: "/shared"},
				},
			},
		},
		RestartPolicy: corev1.RestartPolicyNever,
	}

	jobSpec := batchv1.JobSpec{
		Parallelism:             &listJob.Spec.Parallelism,
		Completions:             &[]int32{int32(len(list))}[0],
		CompletionMode:          func() *batchv1.CompletionMode { mode := batchv1.IndexedCompletion; return &mode }(),
		TTLSecondsAfterFinished: listJob.Spec.TTLSecondsAfterFinished,
		Template: corev1.PodTemplateSpec{
			Spec: podSpec,
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      listJob.Name,
			Namespace: req.Namespace,
		},
		Spec: jobSpec,
	}

	if err := ctrl.SetControllerReference(&listJob, job, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Create(ctx, job); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			log.Error(err, "Failed to create Job")
			return ctrl.Result{}, err
		}
	}

	if listJob.Spec.DeleteAfter != nil {
		return ctrl.Result{RequeueAfter: listJob.Spec.DeleteAfter.Duration}, nil
	}
	return ctrl.Result{}, nil

}

func (r *ListJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchopsv1alpha1.ListJob{}).
		Complete(r)
}
