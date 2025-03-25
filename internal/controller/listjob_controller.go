package controller

import (
	"context"
	"fmt"
	"strings"

	batchopsv1alpha1 "github.com/matanryngler/parallax/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ListJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ListJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling ListJob", "name", req.Name, "namespace", req.Namespace)

	var listJob batchopsv1alpha1.ListJob
	if err := r.Get(ctx, req.NamespacedName, &listJob); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Determine source of list
	var list []string
	if len(listJob.Spec.StaticList) > 0 {
		list = listJob.Spec.StaticList
	} else {
		// Read from ConfigMap generated by ListSource
		var cm corev1.ConfigMap
		err := r.Get(ctx, client.ObjectKey{Name: listJob.Spec.ListSourceRef, Namespace: req.Namespace}, &cm)
		if err != nil {
			return ctrl.Result{}, err
		}
		listStr := cm.Data["items"]
		list = strings.Split(listStr, ",")
	}

	// Create ConfigMap with list
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
		log.Error(err, "failed to create ConfigMap")
		return ctrl.Result{}, client.IgnoreAlreadyExists(err)
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      listJob.Name,
			Namespace: req.Namespace,
		},
		Spec: batchv1.JobSpec{
			Completions: &[]int32{int32(len(list))}[0],
			Parallelism: &listJob.Spec.Parallelism,
			CompletionMode: func() *batchv1.CompletionMode {
				m := batchv1.IndexedCompletion
				return &m
			}(),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
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
							Command: []string{"sh", "-c", "ITEM=$(cut -d',' -f$((`echo $JOB_COMPLETION_INDEX`+1)) /list/items.txt); echo \"export ITEM=$ITEM\" > /shared/env.sh"},
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
							Name:    "main",
							Image:   listJob.Spec.Template.Image,
							Command: []string{"sh", "-c", ". /shared/env.sh && " + strings.Join(listJob.Spec.Template.Command, " ")},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "shared", MountPath: "/shared"},
							},
						},
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(&listJob, job, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Create(ctx, job); err != nil {
		log.Error(err, "failed to create Job")
		return ctrl.Result{}, client.IgnoreAlreadyExists(err)
	}

	return ctrl.Result{}, nil
}

func (r *ListJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchopsv1alpha1.ListJob{}).
		Complete(r)
}
