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

package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	tokenrotatorv1alpha1 "github.com/devonwarren/token-rotator/api/v1alpha1"
	"github.com/devonwarren/token-rotator/internal/rotation"
	"github.com/devonwarren/token-rotator/internal/sources/gitlab"
)

// tokenLifetimeMultiplier is how many rotation intervals the minted GitLab
// token is valid for. Giving the token a lifetime longer than the rotation
// interval means a missed reconcile (crash, pause, skew) won't immediately
// break consumers — the controller gets another chance on the next loop.
const tokenLifetimeMultiplier = 2

// GitLabProjectAccessTokenReconciler reconciles a GitLabProjectAccessToken object
type GitLabProjectAccessTokenReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=token-rotator.org,resources=gitlabprojectaccesstokens,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=token-rotator.org,resources=gitlabprojectaccesstokens/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=token-rotator.org,resources=gitlabprojectaccesstokens/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch

func (r *GitLabProjectAccessTokenReconciler) Reconcile(
	ctx context.Context, req ctrl.Request,
) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var token tokenrotatorv1alpha1.GitLabProjectAccessToken
	if err := r.Get(ctx, req.NamespacedName, &token); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	forceThisReconcile := token.Spec.ForceNow &&
		token.Generation != token.Status.LastForceRotationGeneration

	var lastRotation *time.Time
	if token.Status.LastRotationTime != nil {
		t := token.Status.LastRotationTime.Time
		lastRotation = &t
	}

	decision, err := rotation.Evaluate(
		token.Spec.RotationSchedule, forceThisReconcile, lastRotation, time.Now(),
	)
	if err != nil {
		rotation.SetNotReady(&token.Status.Conditions, token.Generation,
			rotation.ReasonScheduleInvalid, err.Error())
		return ctrl.Result{}, r.updateStatus(ctx, &token)
	}

	if !decision.Due {
		reason := rotation.ReasonWaitingForNext
		if token.Status.LastRotationTime == nil {
			reason = rotation.ReasonNotYetRotated
		}
		rotation.SetReady(&token.Status.Conditions, token.Generation,
			reason, fmt.Sprintf("Next rotation at %s", decision.NextRun.Format(time.RFC3339)))
		token.Status.NextRotationTime = &metav1.Time{Time: decision.NextRun}
		return ctrl.Result{RequeueAfter: time.Until(decision.NextRun)}, r.updateStatus(ctx, &token)
	}

	apiToken, err := r.loadAPIToken(ctx, &token)
	if err != nil {
		log.Error(err, "failed to load GitLab API credential")
		rotation.SetNotReady(&token.Status.Conditions, token.Generation,
			rotation.ReasonMintFailed, err.Error())
		return ctrl.Result{}, r.updateStatus(ctx, &token)
	}

	gitlabClient, err := gitlab.NewClient(apiToken, token.Spec.BaseURL)
	if err != nil {
		rotation.SetNotReady(&token.Status.Conditions, token.Generation,
			rotation.ReasonMintFailed, err.Error())
		return ctrl.Result{}, r.updateStatus(ctx, &token)
	}

	rotationInterval := decision.NextRun.Sub(time.Now())
	if rotationInterval <= 0 {
		rotationInterval = time.Hour
	}
	expiry := time.Now().Add(rotationInterval * tokenLifetimeMultiplier)

	minted, err := gitlabClient.MintProjectAccessToken(ctx, token.Spec, token.Name, expiry)
	if err != nil {
		log.Error(err, "failed to mint GitLab token")
		rotation.SetNotReady(&token.Status.Conditions, token.Generation,
			rotation.ReasonMintFailed, err.Error())
		return ctrl.Result{}, r.updateStatus(ctx, &token)
	}

	secretRef, err := rotation.ExportToSecret(
		ctx, r.Client, &token, r.Scheme, token.Spec.Export, minted.Value,
	)
	if err != nil {
		log.Error(err, "failed to export token to Secret")
		rotation.SetNotReady(&token.Status.Conditions, token.Generation,
			rotation.ReasonExportFailed, err.Error())
		return ctrl.Result{}, r.updateStatus(ctx, &token)
	}

	now := metav1.Now()
	token.Status.LastRotationTime = &now
	token.Status.NextRotationTime = &metav1.Time{Time: decision.NextRun}
	token.Status.CurrentTokenRef = secretRef
	if forceThisReconcile {
		token.Status.LastForceRotationGeneration = token.Generation
	}
	rotation.SetReady(&token.Status.Conditions, token.Generation,
		rotation.ReasonRotated, fmt.Sprintf("Rotated; next rotation at %s",
			decision.NextRun.Format(time.RFC3339)))

	if err := r.updateStatus(ctx, &token); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Until(decision.NextRun)}, nil
}

func (r *GitLabProjectAccessTokenReconciler) updateStatus(
	ctx context.Context, token *tokenrotatorv1alpha1.GitLabProjectAccessToken,
) error {
	if err := r.Status().Update(ctx, token); err != nil && !apierrors.IsConflict(err) {
		return err
	}
	return nil
}

func (r *GitLabProjectAccessTokenReconciler) loadAPIToken(
	ctx context.Context, token *tokenrotatorv1alpha1.GitLabProjectAccessToken,
) (string, error) {
	ref := token.Spec.APITokenSecretRef
	ns := ref.Namespace
	if ns == "" {
		ns = token.Namespace
	}

	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ns}, &secret); err != nil {
		return "", fmt.Errorf("get api token secret %s/%s: %w", ns, ref.Name, err)
	}
	value, ok := secret.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf("secret %s/%s has no key %q", ns, ref.Name, ref.Key)
	}
	if len(value) == 0 {
		return "", fmt.Errorf("secret %s/%s key %q is empty", ns, ref.Name, ref.Key)
	}
	return string(value), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GitLabProjectAccessTokenReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tokenrotatorv1alpha1.GitLabProjectAccessToken{}).
		Owns(&corev1.Secret{}).
		Named("gitlabprojectaccesstoken").
		Complete(r)
}
