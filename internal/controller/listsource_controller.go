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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/PaesslerAG/jsonpath"
	_ "github.com/lib/pq" // PostgreSQL driver
	batchopsv1alpha1 "github.com/matanryngler/parallax/api/v1alpha1"
)

const listSourceFinalizer = "listsource.batchops.io/finalizer"

// ListSourceReconciler reconciles a ListSource object
type ListSourceReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=batchops.io,resources=listsources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batchops.io,resources=listsources/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batchops.io,resources=listsources/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ListSource object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *ListSourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	resourceID := fmt.Sprintf("ListSource/%s.%s", req.Name, req.Namespace)
	log := log.FromContext(ctx).WithValues(
		"resource", resourceID,
	)
	log.Info("Starting reconciliation for ListSource", "resource", resourceID)

	var listSource batchopsv1alpha1.ListSource
	if err := r.Get(ctx, req.NamespacedName, &listSource); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(1).Info("ListSource was not found - it may have been deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Unable to fetch ListSource from the API server")
		return ctrl.Result{}, err
	}

	log = log.WithValues(
		"resourceVersion", listSource.ResourceVersion,
		"generation", listSource.Generation,
		"uid", listSource.UID,
		"type", listSource.Spec.Type,
		"intervalSeconds", listSource.Spec.IntervalSeconds,
	)
	log.V(1).Info("Successfully retrieved ListSource from API server")

	// Always requeue based on interval if specified
	result := ctrl.Result{}
	if listSource.Spec.IntervalSeconds > 0 {
		result.RequeueAfter = time.Duration(listSource.Spec.IntervalSeconds) * time.Second
		log.V(1).Info("Scheduled next reconciliation",
			"next_run_in_seconds", listSource.Spec.IntervalSeconds,
		)
	}

	if !listSource.DeletionTimestamp.IsZero() {
		log.Info("ListSource is being deleted - starting cleanup")
		if controllerutil.ContainsFinalizer(&listSource, listSourceFinalizer) {
			cmID := fmt.Sprintf("ConfigMap/%s.%s", listSource.Name, listSource.Namespace)
			log.Info("Removing associated ConfigMap", "target", cmID)

			cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: listSource.Name, Namespace: listSource.Namespace}}
			if err := r.Delete(ctx, cm); err != nil {
				if !apierrors.IsNotFound(err) {
					log.Error(err, "Failed to remove associated ConfigMap", "target", cmID)
					r.Recorder.Event(&listSource, corev1.EventTypeWarning, "DeleteFailed", fmt.Sprintf("Unable to remove ConfigMap %s: %v", cmID, err))
					return result, err
				}
				log.V(1).Info("Associated ConfigMap was already removed", "target", cmID)
			} else {
				log.Info("Successfully removed associated ConfigMap", "target", cmID)
			}

			controllerutil.RemoveFinalizer(&listSource, listSourceFinalizer)
			if err := r.Update(ctx, &listSource); err != nil {
				log.Error(err, "Unable to remove finalizer from ListSource")
				return result, err
			}
			log.Info("Successfully removed finalizer from ListSource")
		}
		return result, nil
	}

	if !controllerutil.ContainsFinalizer(&listSource, listSourceFinalizer) {
		log.Info("Adding finalizer to ListSource")
		controllerutil.AddFinalizer(&listSource, listSourceFinalizer)
		if err := r.Update(ctx, &listSource); err != nil {
			log.Error(err, "Unable to add finalizer to ListSource")
			return result, err
		}
		log.Info("Successfully added finalizer to ListSource")
		return ctrl.Result{Requeue: true}, nil
	}

	// Get items based on source type
	log.Info("Fetching items from source", "source_type", listSource.Spec.Type)
	items, err := r.getItems(ctx, &listSource)
	if err != nil {
		log.Error(err, "Failed to fetch items from source")
		listSource.Status.Error = err.Error()
		listSource.Status.LastUpdateTime = &metav1.Time{Time: time.Now()}
		if err := r.Status().Update(ctx, &listSource); err != nil {
			log.Error(err, "Unable to update ListSource status with error information")
			return result, err
		}
		r.Recorder.Event(&listSource, corev1.EventTypeWarning, "FetchFailed", fmt.Sprintf("Failed to fetch items: %v", err))
		return result, err
	}
	log.Info("Successfully fetched items from source", "items_found", len(items))

	// Create or update ConfigMap
	cmID := fmt.Sprintf("ConfigMap/%s.%s", listSource.Name, listSource.Namespace)
	cm := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: listSource.Name, Namespace: listSource.Namespace}, cm)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Creating new ConfigMap to store items", "target", cmID)
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      listSource.Name,
					Namespace: listSource.Namespace,
				},
				Data: map[string]string{
					"items": strings.Join(items, ","),
				},
			}
			if err := ctrl.SetControllerReference(&listSource, cm, r.Scheme); err != nil {
				log.Error(err, "Unable to set owner reference on ConfigMap", "target", cmID)
				return result, err
			}
			if err := r.Create(ctx, cm); err != nil {
				log.Error(err, "Failed to create ConfigMap", "target", cmID)
				return result, err
			}
			log.Info("Successfully created ConfigMap with items", "target", cmID)
		} else {
			log.Error(err, "Unable to check if ConfigMap exists", "target", cmID)
			return result, err
		}
	} else {
		// Check if ConfigMap data needs to be updated
		newItemsStr := strings.Join(items, ",")
		if currentItemsStr := cm.Data["items"]; currentItemsStr != newItemsStr {
			log.Info("Updating existing ConfigMap with new items",
				"target", cmID,
				"current_version", cm.ResourceVersion,
				"old_items_count", len(strings.Split(currentItemsStr, ",")),
				"new_items_count", len(items),
			)
			cm.Data = map[string]string{
				"items": newItemsStr,
			}
			if err := r.Update(ctx, cm); err != nil {
				log.Error(err, "Failed to update ConfigMap with new items", "target", cmID)
				return result, err
			}
			log.Info("Successfully updated ConfigMap with new items",
				"target", cmID,
				"new_version", cm.ResourceVersion,
			)
		} else {
			log.V(1).Info("ConfigMap data unchanged, skipping update",
				"target", cmID,
				"items_count", len(items),
			)
		}
	}

	// Update status if needed
	statusChanged := false
	newStatus := batchopsv1alpha1.ListSourceStatus{
		LastUpdateTime: &metav1.Time{Time: time.Now()},
		ItemCount:      len(items),
		Error:          "",
	}

	if listSource.Status.ItemCount != newStatus.ItemCount ||
		listSource.Status.Error != newStatus.Error ||
		listSource.Status.LastUpdateTime == nil ||
		time.Since(listSource.Status.LastUpdateTime.Time) > time.Second {
		statusChanged = true
	}

	if statusChanged {
		oldStatus := listSource.Status.DeepCopy()
		listSource.Status = newStatus

		if err := r.Status().Update(ctx, &listSource); err != nil {
			log.Error(err, "Unable to update ListSource status")
			return result, err
		}
		log.Info("Successfully updated ListSource status",
			"previous_count", oldStatus.ItemCount,
			"new_count", listSource.Status.ItemCount,
			"previous_error", oldStatus.Error,
			"new_version", listSource.ResourceVersion,
		)
		r.Recorder.Event(&listSource, corev1.EventTypeNormal, "Updated", fmt.Sprintf("Successfully updated list with %d items", len(items)))
	} else {
		log.V(1).Info("Status unchanged, skipping update",
			"items_count", len(items),
			"last_update", listSource.Status.LastUpdateTime.Time,
		)
	}

	log.Info("Completed reconciliation successfully", "items_count", len(items))
	return result, nil
}

func (r *ListSourceReconciler) getItems(ctx context.Context, listSource *batchopsv1alpha1.ListSource) ([]string, error) {
	switch listSource.Spec.Type {
	case batchopsv1alpha1.StaticList:
		return listSource.Spec.StaticList, nil
	case batchopsv1alpha1.APIList:
		return r.getItemsFromAPI(ctx, listSource)
	case batchopsv1alpha1.PostgresList:
		return r.getItemsFromPostgres(ctx, listSource.Spec.Postgres, listSource.Namespace)
	default:
		return nil, fmt.Errorf("unsupported list source type: %s", listSource.Spec.Type)
	}
}

func (r *ListSourceReconciler) getItemsFromAPI(ctx context.Context, listSource *batchopsv1alpha1.ListSource) ([]string, error) {
	resourceID := fmt.Sprintf("ListSource/%s.%s", listSource.Name, listSource.Namespace)
	log := log.FromContext(ctx).WithValues(
		"resource", resourceID,
		"type", "api",
		"url", listSource.Spec.API.URL,
		"uid", listSource.UID,
	)
	log.Info("Starting API request to fetch items")

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", listSource.Spec.API.URL, nil)
	if err != nil {
		log.Error(err, "Failed to create HTTP request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for k, v := range listSource.Spec.API.Headers {
		req.Header.Add(k, v)
		log.V(1).Info("Added request header", "header", k)
	}

	// Handle authentication if configured
	if listSource.Spec.API.Auth != nil {
		log.V(1).Info("Setting up authentication for API request", "auth_type", listSource.Spec.API.Auth.Type)
		secret, err := r.getSecret(ctx, listSource.Namespace, listSource.Spec.API.Auth.SecretRef)
		if err != nil {
			log.Error(err, "Failed to retrieve authentication secret")
			return nil, fmt.Errorf("failed to get auth secret: %w", err)
		}

		switch listSource.Spec.API.Auth.Type {
		case batchopsv1alpha1.BasicAuth:
			username := secret[listSource.Spec.API.Auth.UsernameKey]
			req.SetBasicAuth(username, "REDACTED")
			log.V(1).Info("Configured basic auth for request", "username", username)
		case batchopsv1alpha1.BearerAuth:
			req.Header.Set("Authorization", "Bearer REDACTED")
			log.V(1).Info("Configured bearer token authentication for request")
		default:
			log.Error(nil, "Unsupported authentication type specified", "auth_type", listSource.Spec.API.Auth.Type)
			return nil, fmt.Errorf("unsupported auth type: %s", listSource.Spec.API.Auth.Type)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error(err, "API request failed")
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error(nil, "API returned non-200 status code",
			"status_code", resp.StatusCode,
			"status", resp.Status,
		)
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err, "Failed to read API response body")
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON response
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		log.Error(err, "Failed to parse API response as JSON")
		return nil, fmt.Errorf("failed to parse response as JSON: %w", err)
	}

	// Evaluate JSONPath expression
	log.V(1).Info("Extracting items using JSONPath", "expression", listSource.Spec.API.JSONPath)
	eval, err := jsonpath.New(listSource.Spec.API.JSONPath)
	if err != nil {
		log.Error(err, "Invalid JSONPath expression")
		return nil, fmt.Errorf("failed to parse JSONPath expression: %w", err)
	}

	var result interface{}
	if err := eval.Eval(data, &result); err != nil {
		log.Error(err, "Failed to evaluate JSONPath against response")
		return nil, fmt.Errorf("failed to evaluate JSONPath: %w", err)
	}

	// Convert result to []string
	var items []string
	switch v := result.(type) {
	case []interface{}:
		for _, item := range v {
			switch str := item.(type) {
			case string:
				items = append(items, str)
			default:
				items = append(items, fmt.Sprintf("%v", str))
			}
		}
	case []string:
		items = v
	case string:
		items = []string{v}
	default:
		log.Error(nil, "JSONPath result has unexpected type", "type", fmt.Sprintf("%T", result))
		return nil, fmt.Errorf("JSONPath result is not a string or array: %T", result)
	}

	log.Info("Successfully processed API response", "items_found", len(items))
	return items, nil
}

func (r *ListSourceReconciler) getItemsFromPostgres(ctx context.Context, config *batchopsv1alpha1.PostgresConfig, namespace string) ([]string, error) {
	log := log.FromContext(ctx).WithValues(
		"type", "postgresql",
		"namespace", namespace,
		"query", config.Query,
	)
	if config.Auth != nil {
		secretID := fmt.Sprintf("Secret/%s.%s", config.Auth.SecretRef.Name, config.Auth.SecretRef.Namespace)
		log = log.WithValues("auth_secret", secretID)
	}
	log.Info("Starting PostgreSQL query to fetch items")

	// Get secret if specified
	var password string
	if config.Auth != nil {
		log.V(1).Info("Retrieving database credentials")
		secretData, err := r.getSecret(ctx, namespace, config.Auth.SecretRef)
		if err != nil {
			log.Error(err, "Failed to retrieve database credentials")
			return nil, fmt.Errorf("failed to get secret: %w", err)
		}
		password = secretData[config.Auth.PasswordKey]
	}

	// Build connection string
	connStr := config.ConnectionString
	if !strings.Contains(connStr, "password=") && password != "" {
		connStr = fmt.Sprintf("%s password=REDACTED", connStr)
		log.V(1).Info("Added password to connection string")
	}
	if !strings.Contains(connStr, "sslmode=") {
		connStr = fmt.Sprintf("%s sslmode=disable", connStr)
		log.V(1).Info("Set SSL mode to disabled")
	}
	if !strings.Contains(connStr, "connect_timeout=") {
		connStr = fmt.Sprintf("%s connect_timeout=10", connStr)
		log.V(1).Info("Set connection timeout to 10 seconds")
	}

	// Open database connection
	log.V(1).Info("Establishing database connection")
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Error(err, "Failed to establish database connection")
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}
	defer db.Close()

	// Verify connection
	log.V(1).Info("Verifying database connection")
	if err := db.PingContext(ctx); err != nil {
		log.Error(err, "Database connection test failed")
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Execute query
	log.V(1).Info("Executing database query", "query", config.Query)
	rows, err := db.QueryContext(ctx, config.Query)
	if err != nil {
		log.Error(err, "Database query failed")
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Process results
	var items []string
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err != nil {
			log.Error(err, "Failed to read row from query result")
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		log.Error(err, "Error occurred while reading query results")
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	log.Info("Successfully executed database query", "items_found", len(items))
	return items, nil
}

func (r *ListSourceReconciler) getSecret(ctx context.Context, namespace string, ref batchopsv1alpha1.SecretRef) (map[string]string, error) {
	secretNamespace := namespace
	if ref.Namespace != "" {
		secretNamespace = ref.Namespace
	}

	secretID := fmt.Sprintf("Secret/%s.%s", ref.Name, secretNamespace)
	log := log.FromContext(ctx).WithValues("secret", secretID)
	log.V(1).Info("Fetching secret")

	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{Name: ref.Name, Namespace: secretNamespace}, &secret); err != nil {
		log.Error(err, "Failed to get secret")
		return nil, fmt.Errorf("failed to get secret %s: %w", secretID, err)
	}

	log.V(1).Info("Successfully fetched secret")
	secretData := make(map[string]string)
	for k, v := range secret.Data {
		secretData[k] = string(v)
	}

	return secretData, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ListSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("listsource-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchopsv1alpha1.ListSource{}).
		Named("listsource").
		Complete(r)
}
