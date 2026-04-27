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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/devonwarren/token-rotator/api/v1alpha1"
)

// SecretTokenKey is the key inside the exported Secret holding the token value.
const SecretTokenKey = "token"

// ExportToSecret writes (or updates) the target Secret described by the
// ExportSpec with the given token value. If the Secret lives in the same
// namespace as the owning token CR it is set as a controller reference so
// garbage collection runs when the CR is deleted.
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

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      export.Name,
			Namespace: export.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, c, secret, func() error {
		if secret.Annotations == nil && len(export.Annotations) > 0 {
			secret.Annotations = map[string]string{}
		}
		for k, v := range export.Annotations {
			secret.Annotations[k] = v
		}
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
