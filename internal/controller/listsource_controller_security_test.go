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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	batchopsv1alpha1 "github.com/matanryngler/parallax/api/v1alpha1"
)

// TestSQLInjectionProtection tests the SQL injection prevention mechanisms
func TestSQLInjectionProtection(t *testing.T) {
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

	t.Run("Blocks DELETE statement", func(t *testing.T) {
		config := &batchopsv1alpha1.PostgresConfig{
			ConnectionString: "postgresql://test:test@localhost/test",
			Query:            "DELETE FROM users WHERE id = 1",
		}

		items, err := reconciler.getItemsFromPostgres(ctx, config, "default")
		assert.Error(t, err)
		assert.Nil(t, items)
		assert.Contains(t, err.Error(), "forbidden keyword")
		assert.Contains(t, err.Error(), "DELETE")
	})

	t.Run("Blocks DROP statement", func(t *testing.T) {
		config := &batchopsv1alpha1.PostgresConfig{
			ConnectionString: "postgresql://test:test@localhost/test",
			Query:            "DROP TABLE users",
		}

		items, err := reconciler.getItemsFromPostgres(ctx, config, "default")
		assert.Error(t, err)
		assert.Nil(t, items)
		assert.Contains(t, err.Error(), "forbidden keyword")
		assert.Contains(t, err.Error(), "DROP")
	})

	t.Run("Blocks INSERT statement", func(t *testing.T) {
		config := &batchopsv1alpha1.PostgresConfig{
			ConnectionString: "postgresql://test:test@localhost/test",
			Query:            "INSERT INTO users (name) VALUES ('hacker')",
		}

		items, err := reconciler.getItemsFromPostgres(ctx, config, "default")
		assert.Error(t, err)
		assert.Nil(t, items)
		assert.Contains(t, err.Error(), "forbidden keyword")
		assert.Contains(t, err.Error(), "INSERT")
	})

	t.Run("Blocks UPDATE statement", func(t *testing.T) {
		config := &batchopsv1alpha1.PostgresConfig{
			ConnectionString: "postgresql://test:test@localhost/test",
			Query:            "UPDATE users SET role = 'admin' WHERE id = 1",
		}

		items, err := reconciler.getItemsFromPostgres(ctx, config, "default")
		assert.Error(t, err)
		assert.Nil(t, items)
		assert.Contains(t, err.Error(), "forbidden keyword")
		assert.Contains(t, err.Error(), "UPDATE")
	})

	t.Run("Blocks TRUNCATE statement", func(t *testing.T) {
		config := &batchopsv1alpha1.PostgresConfig{
			ConnectionString: "postgresql://test:test@localhost/test",
			Query:            "TRUNCATE TABLE users",
		}

		items, err := reconciler.getItemsFromPostgres(ctx, config, "default")
		assert.Error(t, err)
		assert.Nil(t, items)
		assert.Contains(t, err.Error(), "forbidden keyword")
		assert.Contains(t, err.Error(), "TRUNCATE")
	})

	t.Run("Blocks GRANT statement", func(t *testing.T) {
		config := &batchopsv1alpha1.PostgresConfig{
			ConnectionString: "postgresql://test:test@localhost/test",
			Query:            "GRANT ALL PRIVILEGES ON DATABASE test TO hacker",
		}

		items, err := reconciler.getItemsFromPostgres(ctx, config, "default")
		assert.Error(t, err)
		assert.Nil(t, items)
		assert.Contains(t, err.Error(), "forbidden keyword")
		assert.Contains(t, err.Error(), "GRANT")
	})

	t.Run("Blocks query chaining with semicolons", func(t *testing.T) {
		config := &batchopsv1alpha1.PostgresConfig{
			ConnectionString: "postgresql://test:test@localhost/test",
			// Use only SELECT queries to test semicolon blocking specifically
			Query: "SELECT name FROM users; SELECT id FROM users;",
		}

		items, err := reconciler.getItemsFromPostgres(ctx, config, "default")
		assert.Error(t, err)
		assert.Nil(t, items)
		assert.Contains(t, err.Error(), "semicolon")
	})

	t.Run("Blocks EXEC/EXECUTE statements", func(t *testing.T) {
		config := &batchopsv1alpha1.PostgresConfig{
			ConnectionString: "postgresql://test:test@localhost/test",
			Query:            "EXEC sp_malicious_procedure",
		}

		items, err := reconciler.getItemsFromPostgres(ctx, config, "default")
		assert.Error(t, err)
		assert.Nil(t, items)
		assert.Contains(t, err.Error(), "forbidden keyword")
	})

	t.Run("Blocks ALTER statement", func(t *testing.T) {
		config := &batchopsv1alpha1.PostgresConfig{
			ConnectionString: "postgresql://test:test@localhost/test",
			Query:            "ALTER TABLE users ADD COLUMN malicious_field VARCHAR(255)",
		}

		items, err := reconciler.getItemsFromPostgres(ctx, config, "default")
		assert.Error(t, err)
		assert.Nil(t, items)
		assert.Contains(t, err.Error(), "forbidden keyword")
		assert.Contains(t, err.Error(), "ALTER")
	})

	t.Run("Blocks CREATE statement", func(t *testing.T) {
		config := &batchopsv1alpha1.PostgresConfig{
			ConnectionString: "postgresql://test:test@localhost/test",
			Query:            "CREATE TABLE malicious (id INT)",
		}

		items, err := reconciler.getItemsFromPostgres(ctx, config, "default")
		assert.Error(t, err)
		assert.Nil(t, items)
		assert.Contains(t, err.Error(), "forbidden keyword")
		assert.Contains(t, err.Error(), "CREATE")
	})

	t.Run("Allows valid SELECT with column names containing dangerous keywords", func(t *testing.T) {
		// This test would pass validation but fail at DB connection
		// We're only testing that it passes the SQL validation layer
		config := &batchopsv1alpha1.PostgresConfig{
			ConnectionString: "postgresql://test:test@localhost/test",
			// Column names like "deleted_at", "updated_at" should be allowed
			Query: "SELECT id, name, deleted_at, updated_at FROM users WHERE deleted_at IS NULL",
		}

		// This will fail at DB connection, but should pass SQL validation
		_, err := reconciler.getItemsFromPostgres(ctx, config, "default")
		// We expect a connection error, not a security error
		if err != nil {
			assert.NotContains(t, err.Error(), "forbidden keyword")
			assert.NotContains(t, err.Error(), "semicolon")
		}
	})
}

// TestCrossNamespaceSecretAccess tests that cross-namespace secret access is allowed (controlled by RBAC)
func TestCrossNamespaceSecretAccess(t *testing.T) {
	scheme := runtime.NewScheme()
	err := batchopsv1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	t.Run("Allows cross-namespace secret access when explicitly specified", func(t *testing.T) {
		// Create a secret in production namespace
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "prod-secret",
				Namespace: "production",
			},
			Data: map[string][]byte{
				"password": []byte("prod-password"),
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

		// ListSource in "dev" namespace accesses secret in "production" namespace
		// This should be allowed - RBAC controls the actual access
		secretRef := batchopsv1alpha1.SecretRef{
			Name:      "prod-secret",
			Namespace: "production",
			Key:       "password",
		}

		secretData, err := reconciler.getSecret(context.Background(), "dev", secretRef)
		require.NoError(t, err)
		assert.Equal(t, "prod-password", secretData["password"])
	})

	t.Run("Uses ListSource namespace when secret namespace not specified", func(t *testing.T) {
		// Create a secret in default namespace
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "local-secret",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"apikey": []byte("local-key"),
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

		// No namespace specified - should use ListSource's namespace
		secretRef := batchopsv1alpha1.SecretRef{
			Name: "local-secret",
			Key:  "apikey",
		}

		secretData, err := reconciler.getSecret(context.Background(), "default", secretRef)
		require.NoError(t, err)
		assert.Equal(t, "local-key", secretData["apikey"])
	})
}
