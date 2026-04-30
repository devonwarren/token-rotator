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

package rotation

import (
	"context"
	"errors"
	"fmt"
	"maps"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/devonwarren/token-rotator/api/v1alpha1"
)

// SecretTokenKey is the key inside the exported Secret holding the token value.
const SecretTokenKey = "token"

// ManagedByLabel marks Secrets the controller manages. Its value is the UID
// of the owning token CR. Before writing an export target the controller
// refuses to touch any existing Secret that does not carry this label with
// the expected UID — preventing a CR author from pointing
// spec.export.{namespace,name} at a Secret owned by another controller or
// tenant and causing the controller (which has cluster-wide Secret write)
// to overwrite its contents as a confused deputy.
const ManagedByLabel = "token-rotator.org/managed-by-uid"

// ErrExportTargetConflict is returned when the target Secret already exists
// but was not created by this controller for this owner. The controller
// translates this into a static ExportFailed condition without leaking the
// target Secret's contents.
var ErrExportTargetConflict = errors.New("export target Secret exists and is not owned by this token")

// ExportToSecret writes (or updates) the target Secret described by the
// ExportSpec with the given token value. If the Secret lives in the same
// namespace as the owning token CR it is set as a controller reference so
// garbage collection runs when the CR is deleted. Cross-namespace exports
// skip the owner ref (Kubernetes GC requires same-namespace) but still
// stamp ManagedByLabel so subsequent reconciles can verify ownership.
//
// If the target Secret already exists and either:
//   - carries no ManagedByLabel, or
//   - carries a ManagedByLabel with a different UID, or
//   - is of a Type other than Opaque,
//
// the call refuses with ErrExportTargetConflict. This prevents a CR author
// from hijacking existing Secrets (e.g. service-account tokens, TLS certs,
// other tenants' exported tokens) by naming them as the export target.
func ExportToSecret(
	ctx context.Context,
	c client.Client,
	owner client.Object,
	scheme *runtime.Scheme,
	export v1alpha1.ExportSpec,
	tokenValue string,
) (*corev1.SecretReference, error) {
	if export.Type != v1alpha1.ExportTypeSecret {
		return nil, fmt.Errorf("unsupported export type %q", export.Type)
	}

	ownerUID := string(owner.GetUID())
	if ownerUID == "" {
		return nil, fmt.Errorf("owner has no UID; refusing to export")
	}

	key := types.NamespacedName{Name: export.Name, Namespace: export.Namespace}

	var existing corev1.Secret
	getErr := c.Get(ctx, key, &existing)
	switch {
	case apierrors.IsNotFound(getErr):
		// Fresh target — we'll create it below.
	case getErr != nil:
		return nil, fmt.Errorf("inspect export target: %w", getErr)
	default:
		if existing.Type != "" && existing.Type != corev1.SecretTypeOpaque {
			return nil, ErrExportTargetConflict
		}
		if existing.Labels[ManagedByLabel] != ownerUID {
			return nil, ErrExportTargetConflict
		}
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      export.Name,
			Namespace: export.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, c, secret, func() error {
		if secret.Labels == nil {
			secret.Labels = map[string]string{}
		}
		secret.Labels[ManagedByLabel] = ownerUID

		secret.Annotations = maps.Clone(export.Annotations)

		secret.Type = corev1.SecretTypeOpaque
		if secret.StringData == nil {
			secret.StringData = map[string]string{}
		}
		secret.StringData[SecretTokenKey] = tokenValue

		if owner.GetNamespace() == secret.Namespace {
			return controllerutil.SetControllerReference(owner, secret, scheme)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("upsert secret %s/%s: %w", export.Namespace, export.Name, err)
	}

	return &corev1.SecretReference{Name: secret.Name, Namespace: secret.Namespace}, nil
}
