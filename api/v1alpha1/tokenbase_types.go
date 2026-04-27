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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RotationStrategy controls what happens to the previous token when a new
// one is minted.
// +kubebuilder:validation:Enum=Immediate;KeepOld
type RotationStrategy string

const (
	RotationStrategyImmediate RotationStrategy = "Immediate"
	RotationStrategyKeepOld   RotationStrategy = "KeepOld"
)

// ExportType is the kind of resource the rotated token is written to.
// +kubebuilder:validation:Enum=Secret
type ExportType string

const (
	ExportTypeSecret ExportType = "Secret"
)

// ExportSpec describes where the rotated token is published.
type ExportSpec struct {
	// +required
	Type ExportType `json:"type"`
	// +required
	Name string `json:"name"`
	// +required
	Namespace string `json:"namespace"`
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// TokenSpecBase is embedded in every per-source token CRD.
type TokenSpecBase struct {
	// RotationSchedule is a cron expression controlling when rotations run.
	// +required
	RotationSchedule string `json:"rotationSchedule"`

	// ForceNow triggers a one-shot rotation on the next reconcile, independent
	// of the schedule. The controller clears this after rotating.
	// +optional
	// +kubebuilder:default=false
	ForceNow bool `json:"forceNow,omitempty"`

	// +optional
	// +kubebuilder:default=Immediate
	RotationStrategy RotationStrategy `json:"rotationStrategy,omitempty"`

	// +required
	Export ExportSpec `json:"export"`
}

// TokenStatus is embedded in every per-source token CRD.
type TokenStatus struct {
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	LastRotationTime *metav1.Time `json:"lastRotationTime,omitempty"`

	// +optional
	NextRotationTime *metav1.Time `json:"nextRotationTime,omitempty"`

	// CurrentTokenRef points at the Secret (or other export) holding the
	// currently-active token.
	// +optional
	CurrentTokenRef *corev1.SecretReference `json:"currentTokenRef,omitempty"`

	// PreviousTokenRef is only populated when RotationStrategy=KeepOld and
	// references the Secret holding the prior token during the grace period.
	// +optional
	PreviousTokenRef *corev1.SecretReference `json:"previousTokenRef,omitempty"`
}
