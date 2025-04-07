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
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	batchopsv1alpha1 "github.com/matanryngler/parallax/api/v1alpha1"
)

// Mock database types
type mockRows struct {
	rows   []string
	cursor int
}

func (m *mockRows) Columns() []string {
	return []string{"item"}
}

func (m *mockRows) Close() error {
	return nil
}

func (m *mockRows) Next(dest []driver.Value) error {
	if m.cursor >= len(m.rows) {
		return nil
	}
	dest[0] = m.rows[m.cursor]
	m.cursor++
	return nil
}

type mockConn struct {
	queries map[string][]string // map of query to results
}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	return nil, nil
}

func (c *mockConn) Close() error {
	return nil
}

func (c *mockConn) Begin() (driver.Tx, error) {
	return nil, nil
}

func (c *mockConn) Query(query string, args []driver.Value) (driver.Rows, error) {
	if results, ok := c.queries[query]; ok {
		return &mockRows{rows: results}, nil
	}
	return nil, fmt.Errorf("unexpected query: %s", query)
}

type mockDriver struct{}

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{
		queries: map[string][]string{
			"SELECT fruit_name FROM fruits": {"apple", "banana", "orange"},
			"SELECT * FROM items":           {"item1", "item2", "item3"},
		},
	}, nil
}

func init() {
	sql.Register("mock-postgres", &mockDriver{})
}

func TestListSourceReconciler(t *testing.T) {
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

	// Create manager
	mgr, err := manager.New(cfg, manager.Options{
		Scheme: scheme.Scheme,
	})
	require.NoError(t, err)

	// Create namespace for tests
	k8sClient := mgr.GetClient()
	require.NotNil(t, k8sClient)

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}
	err = k8sClient.Create(ctx, namespace)
	require.NoError(t, err)
	defer k8sClient.Delete(ctx, namespace)

	// Create reconciler
	reconciler := &ListSourceReconciler{
		Client:   k8sClient,
		Scheme:   scheme.Scheme,
		Recorder: mgr.GetEventRecorderFor("listsource-controller"),
	}

	// Set up reconciler with manager
	err = reconciler.SetupWithManager(mgr)
	require.NoError(t, err)

	// Start manager
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		if err := mgr.Start(ctx); err != nil {
			t.Error(err)
		}
	}()

	// Wait for manager to start
	time.Sleep(1 * time.Second)

	// Test basic reconciler setup
	t.Run("reconciler setup", func(t *testing.T) {
		assert.NotNil(t, reconciler)
		assert.NotNil(t, reconciler.Client)
		assert.NotNil(t, reconciler.Scheme)
		assert.NotNil(t, reconciler.Recorder)
	})

	// Setup mock HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authentication
		username, password, ok := r.BasicAuth()
		if !ok || username != "testuser" || password != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Return mock data
		items := []string{"item1", "item2", "item3"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	}))
	defer mockServer.Close()

	t.Run("Static List Source", func(t *testing.T) {
		// Create ListSource
		listSource := &batchopsv1alpha1.ListSource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-static",
				Namespace: namespace.Name,
			},
			Spec: batchopsv1alpha1.ListSourceSpec{
				Type: batchopsv1alpha1.StaticList,
				StaticList: []string{
					"item1",
					"item2",
					"item3",
				},
			},
		}

		err = k8sClient.Create(ctx, listSource)
		require.NoError(t, err)

		// Wait for reconciliation
		time.Sleep(2 * time.Second)

		// Verify ConfigMap was created
		cm := &corev1.ConfigMap{}
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      listSource.Name,
			Namespace: namespace.Name,
		}, cm)
		require.NoError(t, err)
		assert.Equal(t, "item1,item2,item3", cm.Data["items"])

		// Verify ListSource status
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      listSource.Name,
			Namespace: namespace.Name,
		}, listSource)
		require.NoError(t, err)
		assert.Equal(t, 3, listSource.Status.ItemCount)
		assert.Empty(t, listSource.Status.Error)
	})

	t.Run("API List Source", func(t *testing.T) {
		// Create secret for API auth
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "api-secret",
				Namespace: namespace.Name,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"username": []byte("testuser"),
				"password": []byte("testpass"),
			},
		}
		err = k8sClient.Create(ctx, secret)
		require.NoError(t, err)

		// Create ListSource with mock server URL
		listSource := &batchopsv1alpha1.ListSource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-api",
				Namespace: namespace.Name,
			},
			Spec: batchopsv1alpha1.ListSourceSpec{
				Type: batchopsv1alpha1.APIList,
				API: &batchopsv1alpha1.APIConfig{
					URL: mockServer.URL,
					Auth: &batchopsv1alpha1.APIAuth{
						Type: batchopsv1alpha1.BasicAuth,
						SecretRef: batchopsv1alpha1.SecretRef{
							Name: "api-secret",
						},
						UsernameKey: "username",
						PasswordKey: "password",
					},
					JSONPath: "$.items[*].name",
				},
			},
		}

		err = k8sClient.Create(ctx, listSource)
		require.NoError(t, err)

		// Wait for reconciliation
		time.Sleep(2 * time.Second)

		// Verify ConfigMap was created
		cm := &corev1.ConfigMap{}
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      listSource.Name,
			Namespace: namespace.Name,
		}, cm)
		require.NoError(t, err)
		assert.Equal(t, "item1,item2,item3", cm.Data["items"])

		// Verify ListSource status
		err = k8sClient.Get(ctx, client.ObjectKey{
			Name:      listSource.Name,
			Namespace: namespace.Name,
		}, listSource)
		require.NoError(t, err)
		assert.Equal(t, 3, listSource.Status.ItemCount)
		assert.Empty(t, listSource.Status.Error)
	})

	t.Run("PostgreSQL List Source", func(t *testing.T) {
		// Create secret for PostgreSQL auth
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "postgres-secret",
				Namespace: namespace.Name,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"password": []byte("testpass"),
			},
		}
		err = k8sClient.Create(ctx, secret)
		require.NoError(t, err)

		// Wait for secret to be ready
		require.Eventually(t, func() bool {
			var s corev1.Secret
			err := k8sClient.Get(ctx, client.ObjectKey{Name: secret.Name, Namespace: namespace.Name}, &s)
			return err == nil
		}, 5*time.Second, 100*time.Millisecond, "Secret was not created in time")

		// Create ListSource with mock database
		listSource := &batchopsv1alpha1.ListSource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-postgres",
				Namespace: namespace.Name,
			},
			Spec: batchopsv1alpha1.ListSourceSpec{
				Type: batchopsv1alpha1.PostgresList,
				Postgres: &batchopsv1alpha1.PostgresConfig{
					ConnectionString: "host=localhost driver=mock-postgres",
					Query:            "SELECT item FROM items",
					Auth: &batchopsv1alpha1.PostgresAuth{
						SecretRef: batchopsv1alpha1.SecretRef{
							Name:      "postgres-secret",
							Namespace: namespace.Name,
						},
						PasswordKey: "password",
					},
				},
			},
		}

		err = k8sClient.Create(ctx, listSource)
		require.NoError(t, err)

		// Wait for reconciliation to complete and ConfigMap to be created
		require.Eventually(t, func() bool {
			var ls batchopsv1alpha1.ListSource
			err := k8sClient.Get(ctx, client.ObjectKey{Name: listSource.Name, Namespace: namespace.Name}, &ls)
			if err != nil {
				t.Logf("Error getting ListSource: %v", err)
				return false
			}

			if ls.Status.Error != "" {
				t.Logf("ListSource has error: %s", ls.Status.Error)
				return false
			}

			if ls.Status.ItemCount != 3 {
				t.Logf("ListSource has wrong item count: got %d, want 3", ls.Status.ItemCount)
				return false
			}

			var cm corev1.ConfigMap
			err = k8sClient.Get(ctx, client.ObjectKey{Name: listSource.Name, Namespace: namespace.Name}, &cm)
			if err != nil {
				t.Logf("Error getting ConfigMap: %v", err)
				return false
			}

			items := strings.Split(cm.Data["items"], ",")
			if len(items) != 3 {
				t.Logf("ConfigMap has wrong number of items: got %d, want 3", len(items))
				return false
			}

			return true
		}, 10*time.Second, 100*time.Millisecond, "Reconciliation did not complete in time")

		// Verify ConfigMap was created with correct data
		cm := &corev1.ConfigMap{}
		err = k8sClient.Get(ctx, client.ObjectKey{Name: listSource.Name, Namespace: namespace.Name}, cm)
		require.NoError(t, err)
		assert.Equal(t, "item1,item2,item3", cm.Data["items"])

		// Verify ListSource status
		err = k8sClient.Get(ctx, client.ObjectKey{Name: listSource.Name, Namespace: namespace.Name}, listSource)
		require.NoError(t, err)
		assert.Equal(t, 3, listSource.Status.ItemCount)
		assert.Empty(t, listSource.Status.Error)
	})
}

func TestListSourceReconciler_PostgreSQL(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Create expected rows
	rows := sqlmock.NewRows([]string{"item"}).
		AddRow("apple").
		AddRow("banana").
		AddRow("orange")

	// Expect the query to be executed
	mock.ExpectPing() // For the connection test
	mock.ExpectQuery("SELECT fruit_name FROM fruits").WillReturnRows(rows)

	// Create a scheme and fake client
	scheme := runtime.NewScheme()
	_ = batchopsv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Create test secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "postgres-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"password": []byte("test-password"),
		},
	}

	// Create test ListSource
	listSource := &batchopsv1alpha1.ListSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-postgres",
			Namespace: "default",
		},
		Spec: batchopsv1alpha1.ListSourceSpec{
			Type: batchopsv1alpha1.PostgresList,
			Postgres: &batchopsv1alpha1.PostgresConfig{
				ConnectionString: "host=localhost port=5432 user=postgres dbname=testdb",
				Query:            "SELECT fruit_name FROM fruits",
				Auth: &batchopsv1alpha1.PostgresAuth{
					SecretRef: batchopsv1alpha1.SecretRef{
						Name: "postgres-secret",
						Key:  "password",
					},
					PasswordKey: "password",
				},
			},
		},
	}

	// Create the fake client with our objects
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(secret, listSource).
		Build()

	// Create the reconciler with our mocked DB
	reconciler := &ListSourceReconciler{
		Client:   client,
		Scheme:   scheme,
		Recorder: &record.FakeRecorder{},
	}

	// Create a test context with the mock DB
	type dbKeyType string
	const dbKey dbKeyType = "db"
	ctx := context.WithValue(context.Background(), dbKey, db)

	// Test getItemsFromPostgres
	items, err := reconciler.getItemsFromPostgres(ctx, listSource.Spec.Postgres, listSource.Namespace)
	assert.NoError(t, err)
	assert.Equal(t, []string{"apple", "banana", "orange"}, items)

	// Ensure all expectations were met
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
