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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	batchopsv1alpha1 "github.com/matanryngler/parallax/api/v1alpha1"
)

// TestGetItems tests the pure routing logic of getItems function
func TestGetItems_RoutingLogic(t *testing.T) {
	scheme := runtime.NewScheme()
	err := batchopsv1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	recorder := record.NewFakeRecorder(100)

	reconciler := &ListSourceReconciler{
		Client:   fakeClient,
		Scheme:   scheme,
		Recorder: recorder,
	}

	ctx := context.Background()

	t.Run("Static List Source", func(t *testing.T) {
		listSource := &batchopsv1alpha1.ListSource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-static",
				Namespace: "default",
			},
			Spec: batchopsv1alpha1.ListSourceSpec{
				Type:       batchopsv1alpha1.StaticList,
				StaticList: []string{"item1", "item2", "item3"},
			},
		}

		items, err := reconciler.getItems(ctx, listSource)
		require.NoError(t, err)
		assert.Equal(t, []string{"item1", "item2", "item3"}, items)
	})

	t.Run("Unsupported Source Type", func(t *testing.T) {
		listSource := &batchopsv1alpha1.ListSource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-unsupported",
				Namespace: "default",
			},
			Spec: batchopsv1alpha1.ListSourceSpec{
				Type: "unsupported-type",
			},
		}

		items, err := reconciler.getItems(ctx, listSource)
		assert.Error(t, err)
		assert.Nil(t, items)
		assert.Contains(t, err.Error(), "unsupported list source type")
	})
}

// TestGetItemsFromAPI tests the API integration logic with mocked HTTP server
func TestGetItemsFromAPI(t *testing.T) {
	scheme := runtime.NewScheme()
	err := batchopsv1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	t.Run("Successful API Call with JSONPath", func(t *testing.T) {
		// Setup mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"items": []map[string]string{
					{"name": "apple"},
					{"name": "banana"},
					{"name": "orange"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		recorder := record.NewFakeRecorder(100)
		reconciler := &ListSourceReconciler{
			Client:   fakeClient,
			Scheme:   scheme,
			Recorder: recorder,
		}

		listSource := &batchopsv1alpha1.ListSource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-api",
				Namespace: "default",
			},
			Spec: batchopsv1alpha1.ListSourceSpec{
				Type: batchopsv1alpha1.APIList,
				API: &batchopsv1alpha1.APIConfig{
					URL:      server.URL,
					JSONPath: "$.items[*].name",
				},
			},
		}

		items, err := reconciler.getItemsFromAPI(context.Background(), listSource)
		require.NoError(t, err)
		assert.Equal(t, []string{"apple", "banana", "orange"}, items)
	})

	t.Run("API Call with Custom Headers", func(t *testing.T) {
		// Setup mock server that checks headers
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify custom header
			if r.Header.Get("X-Custom-Header") != "test-value" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			response := []string{"header-item1", "header-item2"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		recorder := record.NewFakeRecorder(100)
		reconciler := &ListSourceReconciler{
			Client:   fakeClient,
			Scheme:   scheme,
			Recorder: recorder,
		}

		listSource := &batchopsv1alpha1.ListSource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-api-headers",
				Namespace: "default",
			},
			Spec: batchopsv1alpha1.ListSourceSpec{
				Type: batchopsv1alpha1.APIList,
				API: &batchopsv1alpha1.APIConfig{
					URL: server.URL,
					Headers: map[string]string{
						"X-Custom-Header": "test-value",
					},
					JSONPath: "$[*]",
				},
			},
		}

		items, err := reconciler.getItemsFromAPI(context.Background(), listSource)
		require.NoError(t, err)
		assert.Equal(t, []string{"header-item1", "header-item2"}, items)
	})

	t.Run("API Call with Basic Auth", func(t *testing.T) {
		// Setup mock server that requires auth
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok || username != "testuser" || password != "testpass" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			response := []string{"auth-item1", "auth-item2"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		// Create secret
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "api-secret",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"username": []byte("testuser"),
				"password": []byte("testpass"),
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(secret).
			Build()
		recorder := record.NewFakeRecorder(100)
		reconciler := &ListSourceReconciler{
			Client:   fakeClient,
			Scheme:   scheme,
			Recorder: recorder,
		}

		listSource := &batchopsv1alpha1.ListSource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-api-auth",
				Namespace: "default",
			},
			Spec: batchopsv1alpha1.ListSourceSpec{
				Type: batchopsv1alpha1.APIList,
				API: &batchopsv1alpha1.APIConfig{
					URL: server.URL,
					Auth: &batchopsv1alpha1.APIAuth{
						Type: batchopsv1alpha1.BasicAuth,
						SecretRef: batchopsv1alpha1.SecretRef{
							Name: "api-secret",
						},
						UsernameKey: "username",
						PasswordKey: "password",
					},
					JSONPath: "$[*]",
				},
			},
		}

		items, err := reconciler.getItemsFromAPI(context.Background(), listSource)
		require.NoError(t, err)
		assert.Equal(t, []string{"auth-item1", "auth-item2"}, items)
	})

	t.Run("API Call Handles HTTP Error", func(t *testing.T) {
		// Setup mock server that returns error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		recorder := record.NewFakeRecorder(100)
		reconciler := &ListSourceReconciler{
			Client:   fakeClient,
			Scheme:   scheme,
			Recorder: recorder,
		}

		listSource := &batchopsv1alpha1.ListSource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-api-error",
				Namespace: "default",
			},
			Spec: batchopsv1alpha1.ListSourceSpec{
				Type: batchopsv1alpha1.APIList,
				API: &batchopsv1alpha1.APIConfig{
					URL:      server.URL,
					JSONPath: "$[*]",
				},
			},
		}

		items, err := reconciler.getItemsFromAPI(context.Background(), listSource)
		assert.Error(t, err)
		assert.Nil(t, items)
		assert.Contains(t, err.Error(), "API request failed with status 500")
	})

	t.Run("Invalid JSONPath Expression", func(t *testing.T) {
		// Setup mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{"items": []string{"item1", "item2"}}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		recorder := record.NewFakeRecorder(100)
		reconciler := &ListSourceReconciler{
			Client:   fakeClient,
			Scheme:   scheme,
			Recorder: recorder,
		}

		listSource := &batchopsv1alpha1.ListSource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-api-invalid-jsonpath",
				Namespace: "default",
			},
			Spec: batchopsv1alpha1.ListSourceSpec{
				Type: batchopsv1alpha1.APIList,
				API: &batchopsv1alpha1.APIConfig{
					URL:      server.URL,
					JSONPath: "$.invalid.[*", // Invalid JSONPath
				},
			},
		}

		items, err := reconciler.getItemsFromAPI(context.Background(), listSource)
		assert.Error(t, err)
		assert.Nil(t, items)
		assert.Contains(t, err.Error(), "failed to parse JSONPath expression")
	})
}

// TestGetSecret tests the secret retrieval logic
func TestGetSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	err := batchopsv1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	t.Run("Successfully Retrieve Secret", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"username": []byte("testuser"),
				"password": []byte("testpass"),
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(secret).
			Build()
		recorder := record.NewFakeRecorder(100)
		reconciler := &ListSourceReconciler{
			Client:   fakeClient,
			Scheme:   scheme,
			Recorder: recorder,
		}

		secretRef := batchopsv1alpha1.SecretRef{
			Name: "test-secret",
			Key:  "password",
		}

		secretData, err := reconciler.getSecret(context.Background(), "default", secretRef)
		require.NoError(t, err)
		assert.Equal(t, "testuser", secretData["username"])
		assert.Equal(t, "testpass", secretData["password"])
	})

	t.Run("Secret Not Found", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		recorder := record.NewFakeRecorder(100)
		reconciler := &ListSourceReconciler{
			Client:   fakeClient,
			Scheme:   scheme,
			Recorder: recorder,
		}

		secretRef := batchopsv1alpha1.SecretRef{
			Name: "nonexistent-secret",
			Key:  "password",
		}

		secretData, err := reconciler.getSecret(context.Background(), "default", secretRef)
		assert.Error(t, err)
		assert.Nil(t, secretData)
		assert.Contains(t, err.Error(), "not found")
	})
}
