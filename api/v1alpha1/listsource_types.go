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

type ListSourceType string

const (
	StaticList   ListSourceType = "static"
	APIList      ListSourceType = "api"
	PostgresList ListSourceType = "postgresql"
)

// +kubebuilder:validation:Enum=basic;bearer
type APIAuthType string

const (
	BasicAuth  APIAuthType = "basic"
	BearerAuth APIAuthType = "bearer"
)

type APIConfig struct {
	// +kubebuilder:validation:Required
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Auth    *APIAuth          `json:"auth,omitempty"`
	// +kubebuilder:validation:Required
	JSONPath string `json:"jsonPath,omitempty"`
}

type APIAuth struct {
	// +kubebuilder:validation:Required
	Type APIAuthType `json:"type"`
	// +kubebuilder:validation:Required
	SecretRef SecretRef `json:"secretRef"`
	// +kubebuilder:validation:Required
	UsernameKey string `json:"usernameKey,omitempty"`
	// +kubebuilder:validation:Required
	PasswordKey string `json:"passwordKey,omitempty"`
}

type PostgresConfig struct {
	// +kubebuilder:validation:Required
	ConnectionString string `json:"connectionString"`
	// +kubebuilder:validation:Required
	Query string        `json:"query"`
	Auth  *PostgresAuth `json:"auth,omitempty"`
}

type PostgresAuth struct {
	// +kubebuilder:validation:Required
	SecretRef SecretRef `json:"secretRef"`
	// +kubebuilder:validation:Required
	PasswordKey string `json:"passwordKey"`
}

type SecretRef struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}

type ListSourceSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=static;api;postgresql
	Type ListSourceType `json:"type"`
	// +kubebuilder:validation:Minimum=1
	IntervalSeconds int             `json:"intervalSeconds,omitempty"`
	API             *APIConfig      `json:"api,omitempty"`
	Postgres        *PostgresConfig `json:"postgres,omitempty"`
	StaticList      []string        `json:"staticList,omitempty"`
}

type ListSourceStatus struct {
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`
	ItemCount      int          `json:"itemCount,omitempty"`
	Error          string       `json:"error,omitempty"`
}

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

	Spec   ListSourceSpec   `json:"spec,omitempty"`
	Status ListSourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ListSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ListSource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ListSource{}, &ListSourceList{})
}
