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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	batchopsv1alpha1 "github.com/matanryngler/parallax/api/v1alpha1"
)

func TestListSourceReconcile_StaticList(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, batchopsv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	listSource := &batchopsv1alpha1.ListSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-listsource",
			Namespace: "default",
		},
		Spec: batchopsv1alpha1.ListSourceSpec{
			Type:            batchopsv1alpha1.StaticList,
			StaticList:      []string{"item1", "item2", "item3"},
			IntervalSeconds: 0,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(listSource).
		WithStatusSubresource(listSource).
		Build()

	recorder := record.NewFakeRecorder(100)
	reconciler := &ListSourceReconciler{
		Client:   fakeClient,
		Scheme:   scheme,
		Recorder: recorder,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-listsource",
			Namespace: "default",
		},
	}

	// First reconcile should add finalizer
	result, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.Requeue)

	// Get updated ListSource
	var updatedListSource batchopsv1alpha1.ListSource
	err = fakeClient.Get(ctx, req.NamespacedName, &updatedListSource)
	require.NoError(t, err)
	assert.Contains(t, updatedListSource.Finalizers, listSourceFinalizer)

	// Second reconcile should create ConfigMap
	result, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Verify ConfigMap was created
	var cm corev1.ConfigMap
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-listsource",
		Namespace: "default",
	}, &cm)
	require.NoError(t, err)
	assert.Contains(t, cm.Data["items"], "item1")
	assert.Contains(t, cm.Data["items"], "item2")
	assert.Contains(t, cm.Data["items"], "item3")
}

func TestListSourceReconcile_IntervalRequeue(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, batchopsv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	listSource := &batchopsv1alpha1.ListSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-interval",
			Namespace:  "default",
			Finalizers: []string{listSourceFinalizer},
		},
		Spec: batchopsv1alpha1.ListSourceSpec{
			Type:            batchopsv1alpha1.StaticList,
			StaticList:      []string{"item1"},
			IntervalSeconds: 60,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(listSource).
		WithStatusSubresource(listSource).
		Build()

	recorder := record.NewFakeRecorder(100)
	reconciler := &ListSourceReconciler{
		Client:   fakeClient,
		Scheme:   scheme,
		Recorder: recorder,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-interval",
			Namespace: "default",
		},
	}

	result, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 60*time.Second, result.RequeueAfter)
}

func TestListSourceReconcile_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, batchopsv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	recorder := record.NewFakeRecorder(100)
	reconciler := &ListSourceReconciler{
		Client:   fakeClient,
		Scheme:   scheme,
		Recorder: recorder,
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
	assert.Equal(t, time.Duration(0), result.RequeueAfter)
}

func TestListSourceReconcile_Deletion(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, batchopsv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	now := metav1.NewTime(time.Now())
	listSource := &batchopsv1alpha1.ListSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-deletion",
			Namespace:         "default",
			DeletionTimestamp: &now,
			Finalizers:        []string{listSourceFinalizer},
		},
		Spec: batchopsv1alpha1.ListSourceSpec{
			Type:            batchopsv1alpha1.StaticList,
			StaticList:      []string{"item1"},
			IntervalSeconds: 0,
		},
	}

	// Create ConfigMap that should be cleaned up
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deletion",
			Namespace: "default",
		},
		Data: map[string]string{
			"items": "item1",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(listSource, cm).
		WithStatusSubresource(listSource).
		Build()

	recorder := record.NewFakeRecorder(100)
	reconciler := &ListSourceReconciler{
		Client:   fakeClient,
		Scheme:   scheme,
		Recorder: recorder,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-deletion",
			Namespace: "default",
		},
	}

	// Reconcile should remove finalizer and delete ConfigMap
	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Verify finalizer was removed (resource may be deleted)
	var updatedListSource batchopsv1alpha1.ListSource
	err = fakeClient.Get(ctx, req.NamespacedName, &updatedListSource)
	if err == nil {
		// Resource still exists, check finalizer was removed
		assert.NotContains(t, updatedListSource.Finalizers, listSourceFinalizer)
	}
	// If resource was deleted, that's also a successful outcome
}

func TestListSourceReconcile_EmptyStaticList(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, batchopsv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	listSource := &batchopsv1alpha1.ListSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-empty",
			Namespace:  "default",
			Finalizers: []string{listSourceFinalizer},
		},
		Spec: batchopsv1alpha1.ListSourceSpec{
			Type:            batchopsv1alpha1.StaticList,
			StaticList:      []string{},
			IntervalSeconds: 0,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(listSource).
		WithStatusSubresource(listSource).
		Build()

	recorder := record.NewFakeRecorder(100)
	reconciler := &ListSourceReconciler{
		Client:   fakeClient,
		Scheme:   scheme,
		Recorder: recorder,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-empty",
			Namespace: "default",
		},
	}

	_, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Should still create ConfigMap even with empty list
	var cm corev1.ConfigMap
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-empty",
		Namespace: "default",
	}, &cm)
	require.NoError(t, err)
	assert.NotNil(t, cm.Data)
}

func TestListSourceReconcile_UnsupportedType(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, batchopsv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	listSource := &batchopsv1alpha1.ListSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-unsupported",
			Namespace:  "default",
			Finalizers: []string{listSourceFinalizer},
		},
		Spec: batchopsv1alpha1.ListSourceSpec{
			Type:            "unsupported-type",
			IntervalSeconds: 0,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(listSource).
		WithStatusSubresource(listSource).
		Build()

	recorder := record.NewFakeRecorder(100)
	reconciler := &ListSourceReconciler{
		Client:   fakeClient,
		Scheme:   scheme,
		Recorder: recorder,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-unsupported",
			Namespace: "default",
		},
	}

	// Should return error for unsupported type
	_, err := reconciler.Reconcile(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported list source type")

	// Verify status was updated with error condition
	var updatedListSource batchopsv1alpha1.ListSource
	err = fakeClient.Get(ctx, req.NamespacedName, &updatedListSource)
	require.NoError(t, err)

	// Check for Ready=False condition
	readyCondition := false
	for _, condition := range updatedListSource.Status.Conditions {
		if condition.Type == ConditionTypeReady && condition.Status == metav1.ConditionFalse {
			readyCondition = true
			break
		}
	}
	assert.True(t, readyCondition, "Expected Ready=False condition in status")
}
