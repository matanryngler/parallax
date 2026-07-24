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

func TestListJobReconcile_StaticList(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, batchopsv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	listJob := &batchopsv1alpha1.ListJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-listjob",
			Namespace: "default",
		},
		Spec: batchopsv1alpha1.ListJobSpec{
			StaticList: []string{"item1", "item2", "item3"},
			Template: batchopsv1alpha1.JobTemplateSpec{
				Image:   "alpine:3.19",
				Command: []string{"echo", "test"},
				EnvName: "ITEM",
				Labels: map[string]string{
					"app.kubernetes.io/component": "processor",
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(listJob).
		WithStatusSubresource(listJob).
		Build()

	reconciler := &ListJobReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-listjob",
			Namespace: "default",
		},
	}

	// First reconcile should add finalizer
	result, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.Requeue)

	// Get updated ListJob
	var updatedListJob batchopsv1alpha1.ListJob
	err = fakeClient.Get(ctx, req.NamespacedName, &updatedListJob)
	require.NoError(t, err)
	assert.Contains(t, updatedListJob.Finalizers, listJobFinalizer)

	// Second reconcile should create ConfigMap and Job
	result, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	assert.False(t, result.Requeue)

	// Verify ConfigMap was created
	var cm corev1.ConfigMap
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-listjob-list",
		Namespace: "default",
	}, &cm)
	require.NoError(t, err)
	assert.Contains(t, cm.Data["items"], "item1")
	assert.Contains(t, cm.Data["items"], "item2")
	assert.Contains(t, cm.Data["items"], "item3")

	// Verify Job was created
	var job batchv1.Job
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-listjob",
		Namespace: "default",
	}, &job)
	require.NoError(t, err)
	assert.Equal(t, batchv1.IndexedCompletion, *job.Spec.CompletionMode)
	assert.Equal(t, int32(3), *job.Spec.Completions)
	assert.Equal(t, "test-listjob", job.Labels["listjob.batchops.io/name"])
	assert.Equal(t, "test-listjob", job.Labels["listjob"])
	assert.Equal(t, "test-listjob", job.Spec.Template.Labels["listjob.batchops.io/name"])
	assert.Equal(t, "test-listjob", job.Spec.Template.Labels["listjob"])
	assert.Equal(t, "processor", job.Spec.Template.Labels["app.kubernetes.io/component"])
	assert.Equal(t, []string{"sh", "-c", ". /shared/env.sh && exec \"$@\"", "--", "echo", "test"}, job.Spec.Template.Spec.Containers[0].Command)
}

func TestListJobReconcile_ListSourceRef(t *testing.T) {
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

	listJob := &batchopsv1alpha1.ListJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-listjob-ref",
			Namespace: "default",
		},
		Spec: batchopsv1alpha1.ListJobSpec{
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
		WithObjects(listJob, listSourceCM).
		WithStatusSubresource(listJob).
		Build()

	reconciler := &ListJobReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-listjob-ref",
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

	// Verify Job uses items from ListSource
	var job batchv1.Job
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-listjob-ref",
		Namespace: "default",
	}, &job)
	require.NoError(t, err)
	assert.Equal(t, int32(3), *job.Spec.Completions)
}

func TestListJobReconcile_DeleteAfter(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, batchopsv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	deleteAfter := metav1.Duration{Duration: 1 * time.Nanosecond}
	listJob := &batchopsv1alpha1.ListJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-delete-after",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(time.Now().Add(-1 * time.Hour)),
			Finalizers:        []string{listJobFinalizer},
		},
		Spec: batchopsv1alpha1.ListJobSpec{
			StaticList:  []string{"item1"},
			DeleteAfter: &deleteAfter,
			Template: batchopsv1alpha1.JobTemplateSpec{
				Image:   "alpine:3.19",
				Command: []string{"echo", "test"},
				EnvName: "ITEM",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(listJob).
		WithStatusSubresource(listJob).
		Build()

	reconciler := &ListJobReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-delete-after",
			Namespace: "default",
		},
	}

	// Reconcile should trigger deletion due to expiry
	result, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	assert.False(t, result.Requeue)

	// Verify ListJob is marked for deletion
	var updatedListJob batchopsv1alpha1.ListJob
	err = fakeClient.Get(ctx, req.NamespacedName, &updatedListJob)
	// May be NotFound if deleted, or have DeletionTimestamp set
	if err == nil {
		assert.NotNil(t, updatedListJob.DeletionTimestamp)
	}
}

func TestListJobReconcile_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, batchopsv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	reconciler := &ListJobReconciler{
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

func TestListJobReconcile_Deletion(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, batchopsv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	now := metav1.NewTime(time.Now())
	listJob := &batchopsv1alpha1.ListJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-deletion",
			Namespace:         "default",
			DeletionTimestamp: &now,
			Finalizers:        []string{listJobFinalizer},
		},
		Spec: batchopsv1alpha1.ListJobSpec{
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
		WithObjects(listJob).
		WithStatusSubresource(listJob).
		Build()

	reconciler := &ListJobReconciler{
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
	result, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	assert.False(t, result.Requeue)

	// Verify finalizer was removed (resource may be deleted)
	var updatedListJob batchopsv1alpha1.ListJob
	err = fakeClient.Get(ctx, req.NamespacedName, &updatedListJob)
	if err == nil {
		// Resource still exists, check finalizer was removed
		assert.NotContains(t, updatedListJob.Finalizers, listJobFinalizer)
	}
	// If resource was deleted, that's also a successful outcome
}
