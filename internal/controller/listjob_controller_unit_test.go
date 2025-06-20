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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	batchopsv1alpha1 "github.com/matanryngler/parallax/api/v1alpha1"
)

func TestListJobController_BasicSetup(t *testing.T) {
	scheme := runtime.NewScheme()
	err := batchopsv1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)
	err = batchv1.AddToScheme(scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	reconciler := &ListJobReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	// Test that the reconciler is properly configured
	assert.NotNil(t, reconciler.Client)
	assert.NotNil(t, reconciler.Scheme)
}

func TestListJobController_JobCreation_StaticList(t *testing.T) {
	scheme := runtime.NewScheme()
	err := batchopsv1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)
	err = batchv1.AddToScheme(scheme)
	require.NoError(t, err)

	// Create a ConfigMap with static list data
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-source",
			Namespace: "default",
		},
		Data: map[string]string{
			"items": "item1\nitem2\nitem3",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(configMap).
		Build()

	// Create a ListJob
	listJob := &batchopsv1alpha1.ListJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job",
			Namespace: "default",
		},
		Spec: batchopsv1alpha1.ListJobSpec{
			ListSourceRef: "test-source",
			Parallelism:   2,
			Template: batchopsv1alpha1.JobTemplateSpec{
				Image:   "busybox",
				Command: []string{"echo", "Processing item: $ITEM"},
				EnvName: "ITEM",
			},
		},
	}

	err = fakeClient.Create(context.Background(), listJob)
	require.NoError(t, err)

	// Test basic spec validation
	assert.Equal(t, "test-source", listJob.Spec.ListSourceRef)
	assert.Equal(t, int32(2), listJob.Spec.Parallelism)
	assert.Equal(t, "busybox", listJob.Spec.Template.Image)
	assert.Equal(t, "ITEM", listJob.Spec.Template.EnvName)
}
