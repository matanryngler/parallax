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

	batchopsv1alpha1 "github.com/matanryngler/parallax/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestListCronJobReconciler(t *testing.T) {
	ctx := context.Background()

	// Create test environment
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{"../../config/crd/bases"},
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: getFirstFoundEnvTestBinaryDir(),
	}

	// Start test environment
	cfg, err := testEnv.Start()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	defer testEnv.Stop()

	// Register our custom types with the scheme
	err = batchopsv1alpha1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	// Create a new test client
	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	require.NoError(t, err)
	require.NotNil(t, k8sClient)

	// Create namespace for tests
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}
	err = k8sClient.Create(ctx, namespace)
	require.NoError(t, err)
	defer k8sClient.Delete(ctx, namespace)

	// Setup the reconciler
	reconciler := &ListCronJobReconciler{
		Client: k8sClient,
		Scheme: scheme.Scheme,
	}

	// Test basic reconciler setup
	t.Run("reconciler setup", func(t *testing.T) {
		assert.NotNil(t, reconciler)
		assert.NotNil(t, reconciler.Client)
		assert.NotNil(t, reconciler.Scheme)
	})

	// Add your test cases here
}
