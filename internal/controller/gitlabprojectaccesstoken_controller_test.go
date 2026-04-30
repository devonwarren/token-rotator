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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	tokenrotatorv1alpha1 "github.com/devonwarren/token-rotator/api/v1alpha1"
	"github.com/devonwarren/token-rotator/internal/rotation"
)

const (
	// Intentionally does not match GitLab's glpat- prefix so gitleaks
	// doesn't flag this test fixture as a real leaked token.
	fakeGitLabTokenValue = "fixture-rotated-token-xyz"
	// everyMinuteSchedule fires every minute; a never-rotated CR is always
	// due under it, so tests can trigger the mint path without waiting.
	everyMinuteSchedule = "*/1 * * * *"
)

// newValidToken returns a GitLabProjectAccessToken that passes CRD admission.
// Individual tests mutate it before creating.
func newValidToken(name string) *tokenrotatorv1alpha1.GitLabProjectAccessToken {
	return &tokenrotatorv1alpha1.GitLabProjectAccessToken{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: tokenrotatorv1alpha1.GitLabProjectAccessTokenSpec{
			TokenSpecBase: tokenrotatorv1alpha1.TokenSpecBase{
				RotationSchedule: "0 0 * * *",
				Export: tokenrotatorv1alpha1.ExportSpec{
					Type:      tokenrotatorv1alpha1.ExportTypeSecret,
					Name:      name + "-token",
					Namespace: "default",
				},
			},
			Project:     "group/project",
			AccessLevel: tokenrotatorv1alpha1.GitLabAccessLevelMaintainer,
			Scopes:      []string{"api"},
			APITokenSecretRef: tokenrotatorv1alpha1.SecretKeyRef{
				Name: name + "-api-credential",
				Key:  "token",
			},
		},
	}
}

// fakeGitLabServer stands in for GitLab's /api/v4/projects/:id/access_tokens
// endpoints. It counts CreateProjectAccessToken calls so tests can assert
// whether a reconcile attempted to mint.
type fakeGitLabServer struct {
	*httptest.Server
	createCalls atomic.Int32
}

func newFakeGitLabServer() *fakeGitLabServer {
	f := &fakeGitLabServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/access_tokens"):
			f.createCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":         12345,
				"name":       "test",
				"scopes":     []string{"api"},
				"token":      fakeGitLabTokenValue,
				"expires_at": "2099-01-01",
				"active":     true,
				"revoked":    false,
			})
		default:
			http.NotFound(w, r)
		}
	})
	f.Server = httptest.NewServer(mux)
	return f
}

var _ = Describe("GitLabProjectAccessToken Controller", func() {
	var (
		ctx        context.Context
		reconciler *GitLabProjectAccessTokenReconciler
	)

	BeforeEach(func() {
		ctx = context.Background()
		reconciler = &GitLabProjectAccessTokenReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
	})

	reconcileAndReload := func(token *tokenrotatorv1alpha1.GitLabProjectAccessToken) {
		_, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(token),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(token), token)).To(Succeed())
	}

	expectCondition := func(token *tokenrotatorv1alpha1.GitLabProjectAccessToken, status metav1.ConditionStatus, reason string) {
		cond := meta.FindStatusCondition(token.Status.Conditions, rotation.ConditionReady)
		Expect(cond).NotTo(BeNil(), "Ready condition should be set")
		Expect(cond.Status).To(Equal(status))
		Expect(cond.Reason).To(Equal(reason))
	}

	createCredentialSecret := func(token *tokenrotatorv1alpha1.GitLabProjectAccessToken) {
		GinkgoHelper()
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      token.Spec.APITokenSecretRef.Name,
				Namespace: token.Namespace,
			},
			StringData: map[string]string{token.Spec.APITokenSecretRef.Key: "fake-api-credential"},
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())
		DeferCleanup(k8sClient.Delete, ctx, secret)
	}

	Context("when the rotation schedule is invalid", func() {
		It("marks Ready=False with ScheduleInvalid and does not leak the error into the message", func() {
			token := newValidToken("invalid-schedule")
			token.Spec.RotationSchedule = "definitely-not-cron"
			Expect(k8sClient.Create(ctx, token)).To(Succeed())
			DeferCleanup(k8sClient.Delete, ctx, token)

			reconcileAndReload(token)
			expectCondition(token, metav1.ConditionFalse, rotation.ReasonScheduleInvalid)

			cond := meta.FindStatusCondition(token.Status.Conditions, rotation.ConditionReady)
			Expect(cond.Message).NotTo(ContainSubstring("definitely-not-cron"),
				"status message must not echo user input that originated the parse error")
		})
	})

	Context("when a rotation is not yet due", func() {
		It("marks Ready=True with WaitingForNextRotation and sets NextRotationTime", func() {
			token := newValidToken("not-due")
			Expect(k8sClient.Create(ctx, token)).To(Succeed())
			DeferCleanup(k8sClient.Delete, ctx, token)

			lastRotated := metav1.NewTime(time.Now().Add(-time.Minute))
			token.Status.LastRotationTime = &lastRotated
			Expect(k8sClient.Status().Update(ctx, token)).To(Succeed())

			reconcileAndReload(token)
			expectCondition(token, metav1.ConditionTrue, rotation.ReasonWaitingForNext)
			Expect(token.Status.NextRotationTime).NotTo(BeNil())
			Expect(token.Status.NextRotationTime.Time.After(time.Now())).To(BeTrue())
		})
	})

	Context("when a rotation is due but the API credential Secret is missing", func() {
		It("marks Ready=False with MintFailed", func() {
			token := newValidToken("missing-credential")
			token.Spec.RotationSchedule = everyMinuteSchedule
			Expect(k8sClient.Create(ctx, token)).To(Succeed())
			DeferCleanup(k8sClient.Delete, ctx, token)

			reconcileAndReload(token)
			expectCondition(token, metav1.ConditionFalse, rotation.ReasonMintFailed)
		})
	})

	Context("when a rotation succeeds", func() {
		var fake *fakeGitLabServer

		BeforeEach(func() {
			fake = newFakeGitLabServer()
			DeferCleanup(fake.Close)
		})

		It("mints, exports the token to a Secret, and marks Ready=True", func() {
			token := newValidToken("happy-path")
			token.Spec.RotationSchedule = everyMinuteSchedule
			token.Spec.BaseURL = fake.URL
			token.Spec.Export.Annotations = map[string]string{"role": "deploy-key"}
			Expect(k8sClient.Create(ctx, token)).To(Succeed())
			DeferCleanup(k8sClient.Delete, ctx, token)

			createCredentialSecret(token)
			reconcileAndReload(token)

			expectCondition(token, metav1.ConditionTrue, rotation.ReasonRotated)
			Expect(fake.createCalls.Load()).To(Equal(int32(1)))
			Expect(token.Status.LastRotationTime).NotTo(BeNil())
			Expect(token.Status.CurrentTokenRef).NotTo(BeNil())
			Expect(token.Status.CurrentTokenRef.Name).To(Equal(token.Spec.Export.Name))

			var exported corev1.Secret
			Expect(k8sClient.Get(ctx, client.ObjectKey{
				Name: token.Spec.Export.Name, Namespace: "default",
			}, &exported)).To(Succeed())
			Expect(string(exported.Data[rotation.SecretTokenKey])).To(Equal(fakeGitLabTokenValue))
			Expect(exported.Labels[rotation.ManagedByLabel]).To(Equal(string(token.UID)))
			Expect(exported.Annotations).To(HaveKeyWithValue("role", "deploy-key"))
			DeferCleanup(k8sClient.Delete, ctx, &exported)
		})

		It("refuses to overwrite a foreign Secret and reports ExportFailed", func() {
			foreign := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "foreign-target", Namespace: "default"},
				StringData: map[string]string{"token": "not-ours"},
			}
			Expect(k8sClient.Create(ctx, foreign)).To(Succeed())
			DeferCleanup(k8sClient.Delete, ctx, foreign)

			token := newValidToken("export-conflict")
			token.Spec.RotationSchedule = everyMinuteSchedule
			token.Spec.BaseURL = fake.URL
			token.Spec.Export.Name = foreign.Name
			Expect(k8sClient.Create(ctx, token)).To(Succeed())
			DeferCleanup(k8sClient.Delete, ctx, token)

			createCredentialSecret(token)
			reconcileAndReload(token)

			expectCondition(token, metav1.ConditionFalse, rotation.ReasonExportFailed)
			Expect(fake.createCalls.Load()).To(Equal(int32(1)))

			var stillForeign corev1.Secret
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(foreign), &stillForeign)).To(Succeed())
			Expect(string(stillForeign.Data["token"])).To(Equal("not-ours"))
			cond := meta.FindStatusCondition(token.Status.Conditions, rotation.ConditionReady)
			Expect(cond.Message).NotTo(ContainSubstring(fakeGitLabTokenValue),
				"status message must never embed the rotated token value")
		})
	})

	Context("force rotation gating", func() {
		var fake *fakeGitLabServer

		BeforeEach(func() {
			fake = newFakeGitLabServer()
			DeferCleanup(fake.Close)
		})

		It("rotates once per generation when ForceNow stays true across reconciles", func() {
			token := newValidToken("force-gating")
			token.Spec.BaseURL = fake.URL
			token.Spec.ForceNow = true
			Expect(k8sClient.Create(ctx, token)).To(Succeed())
			DeferCleanup(k8sClient.Delete, ctx, token)
			createCredentialSecret(token)

			reconcileAndReload(token)
			expectCondition(token, metav1.ConditionTrue, rotation.ReasonRotated)
			Expect(fake.createCalls.Load()).To(Equal(int32(1)))
			Expect(token.Status.LastForceRotationGeneration).To(Equal(token.Generation))

			reconcileAndReload(token)
			Expect(fake.createCalls.Load()).To(Equal(int32(1)),
				"second reconcile at same generation must not re-mint")
			expectCondition(token, metav1.ConditionTrue, rotation.ReasonWaitingForNext)

			DeferCleanup(func() {
				exported := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
					Name: token.Spec.Export.Name, Namespace: "default",
				}}
				_ = k8sClient.Delete(ctx, exported)
			})
		})
	})
})
