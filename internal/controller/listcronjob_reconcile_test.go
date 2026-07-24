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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	batchopsv1alpha1 "github.com/matanryngler/parallax/api/v1alpha1"
)

func TestListCronJobReconcile_StaticList(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, batchopsv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	listCronJob := &batchopsv1alpha1.ListCronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-listcronjob",
			Namespace: "default",
		},
		Spec: batchopsv1alpha1.ListCronJobSpec{
			Schedule:   "*/5 * * * *",
			StaticList: []string{"item1", "item2", "item3"},
			Template: batchopsv1alpha1.JobTemplateSpec{
				Image:   "alpine:3.19",
				Command: []string{"echo", "test"},
				EnvName: "ITEM",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(listCronJob).
		WithStatusSubresource(listCronJob).
		Build()

	reconciler := &ListCronJobReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-listcronjob",
			Namespace: "default",
		},
	}

	// First reconcile should add finalizer
	result, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.Requeue)

	// Get updated ListCronJob
	var updatedListCronJob batchopsv1alpha1.ListCronJob
	err = fakeClient.Get(ctx, req.NamespacedName, &updatedListCronJob)
	require.NoError(t, err)
	assert.Contains(t, updatedListCronJob.Finalizers, listCronJobFinalizer)

	// Second reconcile should create ConfigMap and CronJob
	result, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Verify ConfigMap was created
	var cm corev1.ConfigMap
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-listcronjob-list",
		Namespace: "default",
	}, &cm)
	require.NoError(t, err)
	assert.Contains(t, cm.Data["items"], "item1")
	assert.Contains(t, cm.Data["items"], "item2")
	assert.Contains(t, cm.Data["items"], "item3")

	// Verify CronJob was created
	var cronJob batchv1.CronJob
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-listcronjob",
		Namespace: "default",
	}, &cronJob)
	require.NoError(t, err)
	assert.Equal(t, "*/5 * * * *", cronJob.Spec.Schedule)
	assert.Equal(t, batchv1.IndexedCompletion, *cronJob.Spec.JobTemplate.Spec.CompletionMode)
	assert.Equal(t, int32(3), *cronJob.Spec.JobTemplate.Spec.Completions)
	assert.Equal(t, "test-listcronjob", cronJob.Labels["listcronjob.batchops.io/name"])
	assert.Equal(t, "test-listcronjob", cronJob.Labels["listcronjob"])
	assert.Equal(t, "test-listcronjob", cronJob.Spec.JobTemplate.Spec.Template.Labels["listcronjob.batchops.io/name"])
	assert.Equal(t, "test-listcronjob", cronJob.Spec.JobTemplate.Spec.Template.Labels["listcronjob"])
}

func TestListCronJobReconcile_ListSourceRef(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, batchopsv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	// Create ListSource ConfigMap
	listSourceCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-source",
			Namespace: "default",
		},
		Data: map[string]string{
			"items": "alice\nbob\ncharlie",
		},
	}

	listCronJob := &batchopsv1alpha1.ListCronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-listcronjob-ref",
			Namespace: "default",
		},
		Spec: batchopsv1alpha1.ListCronJobSpec{
			Schedule:      "0 * * * *",
			ListSourceRef: "test-source",
			Template: batchopsv1alpha1.JobTemplateSpec{
				Image:   "alpine:3.19",
				Command: []string{"echo", "test"},
				EnvName: "ITEM",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(listCronJob, listSourceCM).
		WithStatusSubresource(listCronJob).
		Build()

	reconciler := &ListCronJobReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-listcronjob-ref",
			Namespace: "default",
		},
	}

	// First reconcile adds finalizer
	result, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.Requeue)

	// Second reconcile creates resources
	result, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Verify CronJob uses items from ListSource
	var cronJob batchv1.CronJob
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-listcronjob-ref",
		Namespace: "default",
	}, &cronJob)
	require.NoError(t, err)
	assert.Equal(t, int32(3), *cronJob.Spec.JobTemplate.Spec.Completions)
}

func TestListCronJobReconcile_ConcurrencyPolicy(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, batchopsv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	concurrencyPolicy := batchv1.ForbidConcurrent
	listCronJob := &batchopsv1alpha1.ListCronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-concurrency",
			Namespace:  "default",
			Finalizers: []string{listCronJobFinalizer},
		},
		Spec: batchopsv1alpha1.ListCronJobSpec{
			Schedule:          "0 0 * * *",
			ConcurrencyPolicy: concurrencyPolicy,
			StaticList:        []string{"item1"},
			Template: batchopsv1alpha1.JobTemplateSpec{
				Image:   "alpine:3.19",
				Command: []string{"echo", "test"},
				EnvName: "ITEM",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(listCronJob).
		WithStatusSubresource(listCronJob).
		Build()

	reconciler := &ListCronJobReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-concurrency",
			Namespace: "default",
		},
	}

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Verify CronJob has correct concurrency policy
	var cronJob batchv1.CronJob
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-concurrency",
		Namespace: "default",
	}, &cronJob)
	require.NoError(t, err)
	assert.Equal(t, batchv1.ForbidConcurrent, cronJob.Spec.ConcurrencyPolicy)
}

func TestListCronJobReconcile_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, batchopsv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	reconciler := &ListCronJobReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "nonexistent",
			Namespace: "default",
		},
	}

	// Should handle not found gracefully
	result, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	assert.False(t, result.Requeue)
}

func TestListCronJobReconcile_Deletion(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, batchopsv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	now := metav1.NewTime(time.Now())
	listCronJob := &batchopsv1alpha1.ListCronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-deletion",
			Namespace:         "default",
			DeletionTimestamp: &now,
			Finalizers:        []string{listCronJobFinalizer},
		},
		Spec: batchopsv1alpha1.ListCronJobSpec{
			Schedule:   "0 * * * *",
			StaticList: []string{"item1"},
			Template: batchopsv1alpha1.JobTemplateSpec{
				Image:   "alpine:3.19",
				Command: []string{"echo", "test"},
				EnvName: "ITEM",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(listCronJob).
		WithStatusSubresource(listCronJob).
		Build()

	reconciler := &ListCronJobReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-deletion",
			Namespace: "default",
		},
	}

	// Reconcile should remove finalizer during deletion
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Verify finalizer was removed (resource may be deleted)
	var updatedListCronJob batchopsv1alpha1.ListCronJob
	err = fakeClient.Get(ctx, req.NamespacedName, &updatedListCronJob)
	if err == nil {
		// Resource still exists, check finalizer was removed
		assert.NotContains(t, updatedListCronJob.Finalizers, listCronJobFinalizer)
	}
	// If resource was deleted, that's also a successful outcome
}

func TestListCronJobReconcile_Suspend(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, batchopsv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	suspend := true
	listCronJob := &batchopsv1alpha1.ListCronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-suspend",
			Namespace:  "default",
			Finalizers: []string{listCronJobFinalizer},
		},
		Spec: batchopsv1alpha1.ListCronJobSpec{
			Schedule:   "0 0 * * *",
			Suspend:    &suspend,
			StaticList: []string{"item1"},
			Template: batchopsv1alpha1.JobTemplateSpec{
				Image:   "alpine:3.19",
				Command: []string{"echo", "test"},
				EnvName: "ITEM",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(listCronJob).
		WithStatusSubresource(listCronJob).
		Build()

	reconciler := &ListCronJobReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-suspend",
			Namespace: "default",
		},
	}

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Verify CronJob is suspended
	var cronJob batchv1.CronJob
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-suspend",
		Namespace: "default",
	}, &cronJob)
	require.NoError(t, err)
	assert.NotNil(t, cronJob.Spec.Suspend)
	assert.True(t, *cronJob.Spec.Suspend)
}
