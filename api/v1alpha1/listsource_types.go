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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListSourceType defines the type of data source for fetching list items.
// Supported types are static (hardcoded list), api (REST API), and postgresql (database query).
type ListSourceType string

const (
	// StaticList represents a hardcoded list of items defined in the spec.
	// Use for testing or when the list is known and unchanging.
	StaticList ListSourceType = "static"

	// APIList fetches items from a REST API endpoint.
	// Supports GET/POST requests with authentication and JSONPath extraction.
	APIList ListSourceType = "api"

	// PostgresList fetches items from a PostgreSQL database query.
	// Supports parameterized queries and connection pooling.
	PostgresList ListSourceType = "postgresql"
)

// APIAuthType defines the authentication method for API requests.
// +kubebuilder:validation:Enum=basic;bearer
type APIAuthType string

const (
	// BasicAuth uses username and password authentication (HTTP Basic Auth).
	BasicAuth APIAuthType = "basic"

	// BearerAuth uses token-based authentication (Authorization: Bearer <token>).
	BearerAuth APIAuthType = "bearer"
)

// APIConfig defines configuration for fetching items from a REST API.
// Supports custom headers, authentication, and JSONPath extraction.
type APIConfig struct {
	// URL is the REST API endpoint to fetch data from.
	// Must be a valid HTTP/HTTPS URL.
	// Example: "https://api.example.com/v1/items"
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// Headers are custom HTTP headers to include in the request.
	// Common uses: Content-Type, Accept, custom auth headers.
	// Example: {"Content-Type": "application/json", "X-API-Version": "v1"}
	// +optional
	Headers map[string]string `json:"headers,omitempty"`

	// Auth specifies authentication configuration for the API request.
	// Supports Basic Auth and Bearer Token authentication.
	// Credentials are stored in Kubernetes Secrets.
	// +optional
	Auth *APIAuth `json:"auth,omitempty"`

	// JSONPath is the expression used to extract items from the API response.
	// Uses standard JSONPath syntax to navigate the JSON structure.
	// Example: "$.items[*].id" extracts all id fields from items array.
	// Example: "$[*].name" extracts name from root array.
	// Required for parsing API responses.
	// +kubebuilder:validation:Required
	JSONPath string `json:"jsonPath,omitempty"`

	// TimeoutSeconds specifies the HTTP request timeout in seconds.
	// Default: 30 seconds if not specified.
	// Range: 1-300 seconds (5 minutes max).
	// Use higher values for slow APIs or large responses.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=300
	// +optional
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`
}

// APIAuth configures authentication for API requests.
// Credentials are retrieved from Kubernetes Secrets for security.
type APIAuth struct {
	// Type specifies the authentication method.
	// Valid values: "basic" (HTTP Basic Auth) or "bearer" (Bearer Token).
	// +kubebuilder:validation:Required
	Type APIAuthType `json:"type"`

	// SecretRef references the Kubernetes Secret containing credentials.
	// The secret must exist in the same namespace as the ListSource (or specified namespace).
	// +kubebuilder:validation:Required
	SecretRef SecretRef `json:"secretRef"`

	// UsernameKey is the key in the Secret containing the username.
	// Required for basic auth, ignored for bearer auth.
	// Example: "username" to read from secret.data.username
	// +optional
	UsernameKey string `json:"usernameKey,omitempty"`

	// PasswordKey is the key in the Secret containing the password or token.
	// Required for basic auth (password), used as token for bearer auth.
	// Example: "password" to read from secret.data.password
	// +optional
	PasswordKey string `json:"passwordKey,omitempty"`
}

// PostgresConfig defines configuration for fetching items from PostgreSQL.
// Supports parameterized queries, connection pooling, and SSL/TLS.
type PostgresConfig struct {
	// ConnectionString is the PostgreSQL connection URL.
	// Format: "postgresql://username:password@host:port/database?options"
	// Example: "postgresql://user:pass@postgres:5432/mydb?sslmode=require"
	// For production, use Auth field instead of embedding credentials in URL.
	// +kubebuilder:validation:Required
	ConnectionString string `json:"connectionString"`

	// Query is the SQL SELECT statement to fetch items.
	// IMPORTANT: Use parameterized queries with $1, $2, etc. to prevent SQL injection.
	// The query should return rows where one column contains the item IDs.
	// Example: "SELECT id FROM orders WHERE status = $1"
	// Example: "SELECT user_id FROM users WHERE active = true"
	// +kubebuilder:validation:Required
	Query string `json:"query"`

	// QueryParams are values for parameterized query placeholders ($1, $2, etc.).
	// Use this with $1, $2, etc. placeholders in the Query field for safe parameterization.
	// Example: If Query is "SELECT id FROM orders WHERE status = $1",
	// QueryParams would be ["pending"] to bind "pending" to $1.
	// +optional
	QueryParams []string `json:"queryParams,omitempty"`

	// Auth specifies authentication configuration for PostgreSQL.
	// Uses Kubernetes Secrets to store credentials securely.
	// If not specified, credentials from ConnectionString are used.
	// +optional
	Auth *PostgresAuth `json:"auth,omitempty"`

	// SSLMode specifies the PostgreSQL SSL/TLS mode.
	// Default: "require" (SSL required, certificate not verified).
	// Valid values:
	//   - disable: No SSL (insecure, not recommended)
	//   - allow: SSL if server supports it
	//   - prefer: Try SSL, fall back to non-SSL
	//   - require: SSL required, certificate not verified
	//   - verify-ca: SSL required, verify server certificate
	//   - verify-full: SSL required, verify certificate and hostname
	// Use "verify-full" for production with proper certificates.
	// +kubebuilder:validation:Enum=disable;allow;prefer;require;verify-ca;verify-full
	// +optional
	SSLMode string `json:"sslMode,omitempty"`
}

// PostgresAuth configures authentication for PostgreSQL connections.
// Credentials are retrieved from Kubernetes Secrets for security.
type PostgresAuth struct {
	// SecretRef references the Kubernetes Secret containing the password.
	// The secret must exist in the same namespace as the ListSource (or specified namespace).
	// +kubebuilder:validation:Required
	SecretRef SecretRef `json:"secretRef"`

	// PasswordKey is the key in the Secret containing the database password.
	// Example: "password" to read from secret.data.password
	// +kubebuilder:validation:Required
	PasswordKey string `json:"passwordKey"`
}

// SecretRef references a key within a Kubernetes Secret.
// Used for securely storing and retrieving credentials.
type SecretRef struct {
	// Name is the name of the Kubernetes Secret.
	// Example: "api-credentials" or "db-password"
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace is the namespace containing the Secret.
	// If not specified, uses the same namespace as the ListSource.
	// Cross-namespace secret access requires appropriate RBAC permissions.
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`

	// Key is the key within the Secret's data field.
	// Example: "token", "password", "username"
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}

// ListSourceSpec defines the desired state of a ListSource.
// Specifies how and where to fetch the list of items to process.
type ListSourceSpec struct {
	// Type specifies the data source type.
	// Must be one of: "static", "api", or "postgresql".
	// This determines which config field (StaticList, API, or Postgres) is used.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=static;api;postgresql
	Type ListSourceType `json:"type"`

	// IntervalSeconds is the refresh interval in seconds.
	// Determines how often the ListSource fetches new data.
	// For static lists, set to 0 (no refresh needed).
	// For API/PostgreSQL, typical values:
	//   - 60: Every minute (frequent updates)
	//   - 300: Every 5 minutes (moderate polling)
	//   - 3600: Every hour (periodic batch)
	// Minimum value: 1 second (if specified).
	// Default: 0 (fetch once, no refresh).
	// +kubebuilder:validation:Minimum=0
	// +optional
	IntervalSeconds int `json:"intervalSeconds,omitempty"`

	// API configuration for REST API data sources.
	// Required when Type is "api", ignored otherwise.
	// Defines the API endpoint, authentication, and item extraction.
	// +optional
	API *APIConfig `json:"api,omitempty"`

	// Postgres configuration for PostgreSQL data sources.
	// Required when Type is "postgresql", ignored otherwise.
	// Defines the database connection and query.
	// +optional
	Postgres *PostgresConfig `json:"postgres,omitempty"`

	// StaticList is a hardcoded list of items.
	// Required when Type is "static", ignored otherwise.
	// Each string becomes an item in the generated ConfigMap.
	// Example: ["item1", "item2", "item3"]
	// +optional
	StaticList []string `json:"staticList,omitempty"`
}

// ListSourceStatus defines the observed state of a ListSource.
// Updated by the controller to reflect the current status of data fetching.
type ListSourceStatus struct {
	// Conditions represent the latest available observations of the ListSource's state.
	// Standard condition types:
	//   - Ready: True if data was successfully fetched and ConfigMap updated
	//   - Error: True if there was an error fetching data
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// LastUpdateTime is the timestamp of the last successful data fetch.
	// Updated each time the ListSource successfully retrieves new data.
	// Use to monitor staleness and detect fetch failures.
	// +optional
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`

	// ItemCount is the number of items in the current list.
	// Reflects the count in the generated ConfigMap.
	// Use to monitor data volume and detect unexpected changes.
	// +optional
	ItemCount int `json:"itemCount,omitempty"`

	// Error contains the error message from the last fetch attempt.
	// Empty if the last fetch was successful.
	// Contains details about API errors, database connection failures, etc.
	// +optional
	Error string `json:"error,omitempty"`

	// State indicates the current operational state.
	// Common values: "Active", "Pending", "Error", "Stale".
	// +optional
	State string `json:"state,omitempty"`
}

// ListSource is a Kubernetes custom resource that fetches and maintains a list of items.
// Items are stored in a ConfigMap and can be consumed by ListJobs for parallel processing.
//
// Supported data sources:
//   - Static: Hardcoded list defined in spec
//   - API: REST API with JSONPath extraction
//   - PostgreSQL: Database query results
//
// The controller automatically refreshes data based on IntervalSeconds and updates
// the associated ConfigMap. ListJobs reference the ConfigMap to get current items.
//
// Example usage:
//
//	apiVersion: batchops.example.com/v1alpha1
//	kind: ListSource
//	metadata:
//	  name: my-items
//	spec:
//	  type: api
//	  intervalSeconds: 300
//	  api:
//	    url: https://api.example.com/items
//	    jsonPath: "$.items[*].id"
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.type"
// +kubebuilder:printcolumn:name="Items",type="integer",JSONPath=".status.itemCount"
// +kubebuilder:printcolumn:name="Last Update",type="date",JSONPath=".status.lastUpdateTime"
// +kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.error"
// +kubebuilder:validation:Required
type ListSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the ListSource.
	// Specifies the data source type and configuration.
	Spec ListSourceSpec `json:"spec,omitempty"`

	// Status represents the observed state of the ListSource.
	// Updated by the controller with fetch results and errors.
	Status ListSourceStatus `json:"status,omitempty"`
}

// ListSourceList contains a list of ListSource resources.
// Used by the Kubernetes API for list operations.
// +kubebuilder:object:root=true
type ListSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of ListSource resources.
	Items []ListSource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ListSource{}, &ListSourceList{})
}
