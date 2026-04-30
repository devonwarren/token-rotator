# Token Rotator — Agent Guide

A Kubernetes operator that mints and rotates API tokens for third-party services (GitLab, Tailscale, Docker Hub, DataDog, ArgoCD, Google Workspace, ...). cert-manager / external-secrets-style project shape.

This file is the single source of truth for both `AGENTS.md` and `CLAUDE.md` (the latter is a symlink). It covers project-specific invariants first; generic kubebuilder reference material is at the bottom.

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

---

# Kubebuilder reference (generic)

Everything below is generic guidance scaffolded by kubebuilder. Project-specific rules above take precedence.

## Project structure

**Single-group layout (default):**
```
cmd/main.go                    Manager entry (registers controllers/webhooks)
api/<version>/*_types.go       CRD schemas (+kubebuilder markers)
api/<version>/zz_generated.*   Auto-generated (DO NOT EDIT)
internal/controller/*          Reconciliation logic
internal/webhook/*             Validation/defaulting (if present)
config/crd/bases/*             Generated CRDs (DO NOT EDIT)
config/rbac/role.yaml          Generated RBAC (DO NOT EDIT)
config/samples/*               Example CRs (edit these)
Makefile                       Build/test/deploy commands
PROJECT                        Kubebuilder metadata Auto-generated (DO NOT EDIT)
```

**Multi-group layout** (for projects with multiple API groups):
```
api/<group>/<version>/*_types.go       CRD schemas by group
internal/controller/<group>/*          Controllers by group
internal/webhook/<group>/<version>/*   Webhooks by group and version (if present)
```

Multi-group layout organizes APIs by group name (e.g., `batch`, `apps`). Check the `PROJECT` file for `multigroup: true`.

**To convert to multi-group layout:**
1. Run: `kubebuilder edit --multigroup=true`
2. Move APIs: `mkdir -p api/<group> && mv api/<version> api/<group>/`
3. Move controllers: `mkdir -p internal/controller/<group> && mv internal/controller/*.go internal/controller/<group>/`
4. Move webhooks (if present): `mkdir -p internal/webhook/<group> && mv internal/webhook/<version> internal/webhook/<group>/`
5. Update import paths in all files
6. Fix `path` in `PROJECT` file for each resource
7. Update test suite CRD paths (add one more `..` to relative paths)

## Critical rules

### Never edit these (auto-generated)
- `config/crd/bases/*.yaml` - from `make manifests`
- `config/rbac/role.yaml` - from `make manifests`
- `config/webhook/manifests.yaml` - from `make manifests`
- `**/zz_generated.*.go` - from `make generate`
- `PROJECT` - from `kubebuilder [OPTIONS]`

### Never remove scaffold markers
Do NOT delete `// +kubebuilder:scaffold:*` comments. CLI injects code at these markers.

### Keep project structure
Do not move files around. The CLI expects files in specific locations.

### Always use CLI commands
Always use `kubebuilder create api` and `kubebuilder create webhook` to scaffold. Do NOT create files manually.

### E2E tests require an isolated Kind cluster
The e2e tests are designed to validate the solution in an isolated environment (similar to GitHub Actions CI).
Ensure you run them against a dedicated [Kind](https://kind.sigs.k8s.io/) cluster (not your "real" dev/prod cluster).

## After making changes

**After editing `*_types.go` or markers:**
```
make manifests  # Regenerate CRDs/RBAC from markers
make generate   # Regenerate DeepCopy methods
```

**After editing `*.go` files:**
```
make lint-fix   # Auto-fix code style
make test       # Run unit tests
```

## CLI commands cheat sheet

### Create API (your own types)
```bash
kubebuilder create api --group <group> --version <version> --kind <Kind>
```

### Deploy Image Plugin (scaffold to deploy/manage ANY container image)

Generate a controller that deploys and manages a container image (nginx, redis, memcached, your app, etc.):

```bash
# Example: deploying memcached
kubebuilder create api --group example.com --version v1alpha1 --kind Memcached \
  --image=memcached:alpine \
  --plugins=deploy-image.go.kubebuilder.io/v1-alpha
```

Scaffolds good-practice code: reconciliation logic, status conditions, finalizers, RBAC. Use as a reference implementation.

### Create webhooks
```bash
# Validation + defaulting
kubebuilder create webhook --group <group> --version <version> --kind <Kind> \
  --defaulting --programmatic-validation

# Conversion webhook (for multi-version APIs)
kubebuilder create webhook --group <group> --version v1 --kind <Kind> \
  --conversion --spoke v2
```

### Controller for core Kubernetes types
```bash
# Watch Pods
kubebuilder create api --group core --version v1 --kind Pod \
  --controller=true --resource=false

# Watch Deployments
kubebuilder create api --group apps --version v1 --kind Deployment \
  --controller=true --resource=false
```

### Controller for external types (e.g., from other operators)

Watch resources from external APIs (cert-manager, Argo CD, Istio, etc.):

```bash
# Example: watching cert-manager Certificate resources
kubebuilder create api \
  --group cert-manager --version v1 --kind Certificate \
  --controller=true --resource=false \
  --external-api-path=github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1 \
  --external-api-domain=io \
  --external-api-module=github.com/cert-manager/cert-manager
```

**Note:** Use `--external-api-module=<module>@<version>` only if you need a specific version. Otherwise, omit `@<version>` to use what's in go.mod.

### Webhook for external types

```bash
# Example: validating external resources
kubebuilder create webhook \
  --group cert-manager --version v1 --kind Issuer \
  --defaulting \
  --external-api-path=github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1 \
  --external-api-domain=io \
  --external-api-module=github.com/cert-manager/cert-manager
```

## Testing & development

```bash
make test              # Run unit tests (uses envtest: real K8s API + etcd)
make run               # Run locally (uses current kubeconfig context)
```

Tests use **Ginkgo + Gomega** (BDD style). Check `suite_test.go` for setup.

## Deployment workflow

```bash
# 1. Regenerate manifests
make manifests generate

# 2. Build & deploy
export IMG=<registry>/<project>:tag
make docker-build docker-push IMG=$IMG  # Or: kind load docker-image $IMG --name <cluster>
make deploy IMG=$IMG

# 3. Test
kubectl apply -k config/samples/

# 4. Debug
kubectl logs -n <project>-system deployment/<project>-controller-manager -c manager -f
```

### API design

**Key markers for** `api/<version>/*_types.go`:

```go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=".status.conditions[?(@.type=='Ready')].status"

// On fields:
// +kubebuilder:validation:Required
// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:MaxLength=100
// +kubebuilder:validation:Pattern="^[a-z]+$"
// +kubebuilder:default="value"
```

- **Use** `metav1.Condition` for status (not custom string fields)
- **Use predefined types**: `metav1.Time` instead of `string` for dates
- **Follow K8s API conventions**: Standard field names (`spec`, `status`, `metadata`)

### Controller design

**RBAC markers in** `internal/controller/*_controller.go`:

```go
// +kubebuilder:rbac:groups=mygroup.example.com,resources=mykinds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mygroup.example.com,resources=mykinds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mygroup.example.com,resources=mykinds/finalizers,verbs=update
// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
```

**Implementation rules:**
- **Idempotent reconciliation**: Safe to run multiple times
- **Re-fetch before updates**: `r.Get(ctx, req.NamespacedName, obj)` before `r.Update` to avoid conflicts
- **Structured logging**: `log := log.FromContext(ctx); log.Info("msg", "key", val)`
- **Owner references**: Enable automatic garbage collection (`SetControllerReference`)
- **Watch secondary resources**: Use `.Owns()` or `.Watches()`, not just `RequeueAfter`
- **Finalizers**: Clean up external resources (buckets, VMs, DNS entries)

### Logging

**Follow Kubernetes logging message style guidelines:**

- Start from a capital letter
- Do not end the message with a period
- Active voice: subject present (`"Deployment could not create Pod"`) or omitted (`"Could not create Pod"`)
- Past tense: `"Could not delete Pod"` not `"Cannot delete Pod"`
- Specify object type: `"Deleted Pod"` not `"Deleted"`
- Balanced key-value pairs

```go
log.Info("Starting reconciliation")
log.Info("Created Deployment", "name", deploy.Name)
log.Error(err, "Failed to create Pod", "name", name)
```

**Reference:** https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md#message-style-guidelines

### Webhooks
- **Create all types together**: `--defaulting --programmatic-validation --conversion`
- **When `--force` is used**: Backup custom logic first, then restore after scaffolding
- **For multi-version APIs**: Use hub-and-spoke pattern (`--conversion --spoke v2`)
  - Hub version: Usually oldest stable version (v1)
  - Spoke versions: Newer versions that convert to/from hub (v2, v3)
  - Example: `--group crew --version v1 --kind Captain --conversion --spoke v2` (v1 is hub, v2 is spoke)

### Learning from examples

The **deploy-image plugin** scaffolds a complete controller following good practices. Use it as a reference implementation:

```bash
kubebuilder create api --group example --version v1alpha1 --kind MyApp \
  --image=<your-image> --plugins=deploy-image.go.kubebuilder.io/v1-alpha
```

Generated code includes: status conditions (`metav1.Condition`), finalizers, owner references, events, idempotent reconciliation.

## Distribution options

### Option 1: YAML bundle (Kustomize)

```bash
# Generate dist/install.yaml from Kustomize manifests
make build-installer IMG=<registry>/<project>:tag
```

**Key points:**
- The `dist/install.yaml` is generated from Kustomize manifests (CRDs, RBAC, Deployment)
- Commit this file to your repository for easy distribution
- Users only need `kubectl` to install (no additional tools required)

**Example:** Users install with a single command:
```bash
kubectl apply -f https://raw.githubusercontent.com/<org>/<repo>/<tag>/dist/install.yaml
```

### Option 2: Helm chart

```bash
kubebuilder edit --plugins=helm/v2-alpha                      # Generates dist/chart/ (default)
kubebuilder edit --plugins=helm/v2-alpha --output-dir=charts  # Generates charts/chart/
```

**For development:**
```bash
make helm-deploy IMG=<registry>/<project>:<tag>          # Deploy manager via Helm
make helm-deploy IMG=$IMG HELM_EXTRA_ARGS="--set ..."    # Deploy with custom values
make helm-status                                         # Show release status
make helm-uninstall                                      # Remove release
make helm-history                                        # View release history
make helm-rollback                                       # Rollback to previous version
```

**For end users/production:**
```bash
helm install my-release ./<output-dir>/chart/ --namespace <ns> --create-namespace
```

**Important:** If you add webhooks or modify manifests after initial chart generation:
1. Backup any customizations in `<output-dir>/chart/values.yaml` and `<output-dir>/chart/manager/manager.yaml`
2. Re-run: `kubebuilder edit --plugins=helm/v2-alpha --force` (use same `--output-dir` if customized)
3. Manually restore your custom values from the backup

### Publish container image

```bash
export IMG=<registry>/<project>:<version>
make docker-build docker-push IMG=$IMG
```

## References

### Essential reading
- **Kubebuilder Book**: https://book.kubebuilder.io (comprehensive guide)
- **controller-runtime FAQ**: https://github.com/kubernetes-sigs/controller-runtime/blob/main/FAQ.md (common patterns and questions)
- **Good Practices**: https://book.kubebuilder.io/reference/good-practices.html (why reconciliation is idempotent, status conditions, etc.)
- **Logging Conventions**: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md#message-style-guidelines (message style, verbosity levels)

### API design & implementation
- **API Conventions**: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md
- **Operator Pattern**: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
- **Markers Reference**: https://book.kubebuilder.io/reference/markers.html

### Tools & libraries
- **controller-runtime**: https://github.com/kubernetes-sigs/controller-runtime
- **controller-tools**: https://github.com/kubernetes-sigs/controller-tools
- **Kubebuilder Repo**: https://github.com/kubernetes-sigs/kubebuilder
