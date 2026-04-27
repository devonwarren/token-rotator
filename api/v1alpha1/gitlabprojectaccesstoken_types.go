/*
Copyright 2026 Devon Warren.

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

// GitLabAccessLevel maps to GitLab's numeric access_level values used when
// minting project access tokens.
// +kubebuilder:validation:Enum=Guest;Reporter;Developer;Maintainer;Owner
type GitLabAccessLevel string

const (
	GitLabAccessLevelGuest      GitLabAccessLevel = "Guest"
	GitLabAccessLevelReporter   GitLabAccessLevel = "Reporter"
	GitLabAccessLevelDeveloper  GitLabAccessLevel = "Developer"
	GitLabAccessLevelMaintainer GitLabAccessLevel = "Maintainer"
	GitLabAccessLevelOwner      GitLabAccessLevel = "Owner"
)

// GitLabProjectAccessTokenSpec defines the desired state of a GitLab
// project-scoped access token.
type GitLabProjectAccessTokenSpec struct {
	TokenSpecBase `json:",inline"`

	// Project is the full path (e.g. "mygroup/myproject") or numeric ID of
	// the GitLab project the token is scoped to.
	// +required
	Project string `json:"project"`

	// +required
	AccessLevel GitLabAccessLevel `json:"accessLevel"`

	// Scopes are GitLab PAT scopes like "api", "read_repository", etc.
	// +required
	// +kubebuilder:validation:MinItems=1
	Scopes []string `json:"scopes"`

	// BaseURL overrides the GitLab API endpoint for self-hosted instances.
	// Defaults to https://gitlab.com when unset.
	// +optional
	BaseURL string `json:"baseURL,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories={tokens,gitlab}
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Last Rotated",type=date,JSONPath=`.status.lastRotationTime`
// +kubebuilder:printcolumn:name="Next Rotation",type=date,JSONPath=`.status.nextRotationTime`
// +kubebuilder:printcolumn:name="Export",type=string,JSONPath=`.spec.export.name`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Project",type=string,priority=1,JSONPath=`.spec.project`
// +kubebuilder:printcolumn:name="Access Level",type=string,priority=1,JSONPath=`.spec.accessLevel`

// GitLabProjectAccessToken is the Schema for the gitlabprojectaccesstokens API
type GitLabProjectAccessToken struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of GitLabProjectAccessToken
	// +required
	Spec GitLabProjectAccessTokenSpec `json:"spec"`

	// status defines the observed state of GitLabProjectAccessToken
	// +optional
	Status TokenStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// GitLabProjectAccessTokenList contains a list of GitLabProjectAccessToken
type GitLabProjectAccessTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []GitLabProjectAccessToken `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GitLabProjectAccessToken{}, &GitLabProjectAccessTokenList{})
}
