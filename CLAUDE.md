# Token Rotator — CLAUDE.md

A Kubernetes operator that mints and rotates API tokens for third-party services (GitLab, Tailscale, Docker Hub, DataDog, ArgoCD, Google Workspace, ...). cert-manager / external-secrets-style project shape.

Read `AGENTS.md` for the generic kubebuilder guidance (what files are auto-generated, which markers matter, how to regenerate). This file is everything *specific* to this project.

## Project invariants

These were deliberated and should not be changed without discussion.

- **One strongly-typed CRD per token source.** A single generic `Token` CRD with an opaque config field was explicitly considered and rejected — the scope/target/expiry shapes across providers are too different to fit a single schema usefully. A generic `CustomToken` escape-hatch CRD may be added later but is not the default. See `README.md` ("One CRD per token type").
- **Single API group: `token-rotator.org`.** Kinds are source-prefixed (`GitLabProjectAccessToken`, `TailscaleAuthKey`). Follows the cert-manager / ESO / ArgoCD convention. If the project ever grows to Crossplane scale (dozens of kinds per source) the trade-off to per-source subgroups could be revisited, but we're not close.
- **Shared `TokenSpecBase` and `TokenStatus` embedded in every source CRD.** Common scheduling/export/status fields live there. Per-source fields are added at the subclass. See `api/v1alpha1/tokenbase_types.go`.
- **Shared category registration.** Every source CRD joins `tokens` (so `kubectl get tokens` lists all of them) and its own source category (e.g. `gitlab`). Baseline printer columns — Ready, Last Rotated, Next Rotation, Export, Age — are consistent; source-specific columns go to `priority: 1`.
- **No `TokenSource` interface.** The specs vary too much to abstract usefully. Each per-source reconciler calls its concrete client directly; the shared behavior is plain helper functions in `internal/rotation/`.
- **Per-CR credential.** Each token CR references its own API credential via `spec.apiTokenSecretRef` — the controller loads it at reconcile time. An ESO-style central `Provider` CRD was considered and deferred until there are more sources.

## Security posture

This is a secrets-handling operator — the controller holds credentials that can mint more credentials. Treat accordingly.

**Supply-chain protections in CI:**
- All GitHub Actions pinned by commit SHA (tag comment only). Never pin by tag — mutable refs have been weaponized (tj-actions/changed-files, March 2025).
- `govulncheck` (Go call-graph vuln scanner), `osv-scanner` (broader, covers Actions and `go.sum`), and `gitleaks` (secret scanner) all run on every PR + weekly cron.
- Published images are cosign-signed (keyless / OIDC), have SPDX SBOM attestations attached, and SLSA build-provenance attestations.
- Dependabot is grouped weekly for gomod (k8s/*, sigs.k8s.io/*, gitlab-org/*), github-actions, and the Dockerfile.
- Manager pod runs `runAsNonRoot`, `readOnlyRootFilesystem`, `allowPrivilegeEscalation: false`, drops all capabilities, uses `seccompProfile: RuntimeDefault`.

**In-code discipline:**
- **Never log or surface `minted.Value`** (the rotated token) or the API credential loaded from a Secret. Errors are propagated but must not embed the token value. Status/conditions/events must not contain either.
- Cross-namespace Secret export loses the owner-reference garbage-collection guarantee — `internal/rotation/export.go` intentionally skips the owner ref when `owner.Namespace != secret.Namespace`. If you change this, confirm the GC implications.
- Token lifetime is `2 * rotationInterval`. This gives a missed reconcile a second chance before consumers break. Don't collapse this to `1 * rotationInterval` without a replacement mitigation for reconciler crashes.

## Commit and branching conventions

- Branch for the Go implementation: `migrate-to-go` (not yet merged). Python history preserved on `python-archive` (remote).
- Commits are **imperative present-tense lowercase subject**, grouped by logical change. Examples from the Go migration:
  - `scaffold kubebuilder v4 project with GitLabProjectAccessToken CRD`
  - `define TokenSpecBase and GitLabProjectAccessToken fields`
  - `implement GitLabProjectAccessToken reconciler`
  - `harden supply chain: pin actions, sign images, scan deps and secrets`
- Every commit has a `Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>` trailer when Claude contributed.
- Never `git push` without explicit confirmation from the user (repo-wide rule from user's global CLAUDE.md).

## Local development

```sh
make manifests generate   # after editing *_types.go
make lint-fix             # gofmt + goimports + modernize fixes
make build                # sanity check
make test                 # unit + envtest
make run                  # run controller against current kubeconfig
```

The repo's `make lint` uses a custom golangci-lint config with logcheck enabled; `make lint-fix` applies most findings automatically. Keep lint clean before committing.

## Where things are

```
api/v1alpha1/
  tokenbase_types.go              shared spec/status embedded by every source CRD
  gitlabprojectaccesstoken_types.go
  zz_generated.deepcopy.go        DO NOT EDIT

internal/
  controller/                     per-CRD reconcilers (one file per CRD kind)
  rotation/                       shared helpers — schedule, conditions, export
  sources/<source>/               per-source API clients (thin wrappers)

config/
  crd/bases/                      generated CRDs — DO NOT EDIT
  manager/manager.yaml            manager Deployment (securely hardened)
  rbac/role.yaml                  generated — DO NOT EDIT
  samples/                        example CRs
```

## Open design decisions (from README)

Still TBD at the time of writing:
- Failure handling mechanism — Prometheus metrics, Argo Notifications, status-only, or some combination?
- `RotationStrategy: KeepOld` grace period — how long to retain and what triggers revocation of the old token?
- Additional export targets beyond Kubernetes `Secret` — ESO PushSecret, webhook, …?
- Managing the controller's own API credentials via a self-referential CRD?

If any of these come up, they warrant design discussion with the user before implementation.
