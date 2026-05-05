# Resolutions to README Open Design Questions

Working plan for the four open questions in `README.md`. Each section captures
the decision, rationale, and the concrete work implied.

---

## 1. Failure handling: observability surface

**Decision:** Emit Kubernetes Events for rotation lifecycle, plus metrics via the
OpenTelemetry SDK configured to export to both Prometheus (scrape endpoint) and
OTLP (push).

**Rationale:**
- Kubernetes Events are idiomatic, cheap, and the integration point every
  downstream tool (event-exporter, Argo Events, Argo Notifications,
  Alertmanager) already consumes. Covers "someone should know."
- Prometheus is where the kubebuilder / operator ecosystem lives; every peer
  project (cert-manager, ESO, ArgoCD, Flux) uses it. Users' dashboards and
  Alertmanager rules transfer with no effort.
- OTel API in code lets the same instrumentation also push OTLP for users on
  OTel-native stacks, and leaves the door open for traces later without a
  second instrumentation system. Cost is extra `main.go` plumbing; benefit is
  not having to migrate later.
- Labels/attributes only carry names and reasons — never token values — so this
  conforms to the existing "never log `minted.Value`" posture.

**Scope of work:**
- Wire the OTel SDK in `cmd/main.go`. Prometheus exporter mounted on the
  existing controller-runtime metrics server; OTLP exporter configurable via
  env vars (off by default).
- Instruments:
  - `token_rotator_rotation_attempts_total{kind,source,namespace,name,result}`
    (counter; `result` ∈ `success|failure`).
  - `token_rotator_rotation_failures_total{kind,source,namespace,name,reason}`
    (counter; derived/convenience).
  - `token_rotator_last_success_timestamp_seconds{...}` (async gauge; callback
    reads informer cache).
  - `token_rotator_token_expiry_timestamp_seconds{...}` (async gauge).
- Cardinality: per-CR labels (`namespace`, `name`). Matches cert-manager.
- Event reasons (emit via `record.EventRecorder`):
  `RotationStarted`, `RotationSucceeded`, `RotationFailed`,
  `TokenRevoked`, `ExportUpdated`, `TookOwnership`,
  `DependencyCycle`, `SecretNotAdopted`, `InvalidGracePeriod`.
- Status conditions continue to surface `Ready` with reason/message as today.

**Explicitly not doing:** bespoke webhook notifications from the operator,
Argo Notifications-style annotation templates on the CR.

---

## 2. `KeepOld` grace period

**Decision:** Fixed-duration grace period on the CR. Default `1h`. Ceiling is
`min(7d, rotationInterval - ε)` — the `< rotationInterval` constraint
guarantees at most one extra token exists at any moment.

**Schema:**
```yaml
spec:
  rotationStrategy:
    type: KeepOld
    keepOld:
      gracePeriod: 1h
```

**Controller behavior:**
- On successful mint + export, record `status.previousTokenRef` and
  `status.previousTokenRevocationTime = now + gracePeriod`. Requeue at that
  time.
- On reconcile after the deadline, call the source's revoke API (idempotent —
  treat 404 as success) and clear `previousTokenRef`.
- Validate `gracePeriod < rotationInterval` at reconcile; on violation set
  `Ready=False, reason=InvalidGracePeriod`. (Can't express dynamically in CRD
  OpenAPI validation.)
- Hard cap enforced via CRD validation: `gracePeriod <= 7d`.
- Crash recovery: status persists the revocation deadline, so a restarted
  controller resumes correctly.

**Edge cases:**
- **Rotation N+1 fails while the previous token's grace window is still open:**
  keep the previous-token revocation on its original schedule. Don't extend
  (the previous one might be the leaked credential).
- **New token minted but export fails:** revoke the newly-minted token
  immediately on detection — don't leak a third credential into the wild.

**Status surface:**
- `status.previousTokenRef` and `status.previousTokenRevocationTime`.
- Printer column showing the grace-window countdown so the two-token state is
  visible in `kubectl get`.

**User guidance in docs:** pair with Reloader (in-cluster) or ESO's
refresh-interval tuning (external) if pods/consumers are slow to pick up the
new Secret. The operator's contract ends at "Secret contains current value."

---

## 3. Export targets

**Decision:** `Secret` only. Keep the existing single `export` block — do not
migrate to `exports[]` until a second target is actually needed.

**Rationale:**
- ESO's `PushSecret` is a standalone resource that references any Secret via
  `spec.selector.secret.name`. Users compose it on top of our Secret output
  without any integration work from us.
- ESO's Webhook provider (SecretStore with `provider: webhook`) handles
  webhook delivery end-to-end: URL, auth, body templating, retries. Better at
  webhooks than we would ever be, with none of the security surface.
- YAGNI on `exports[]`: v1alpha1 allows breaking changes, so we can migrate
  when a real second target appears.

**Composition patterns to document** (new README section):
- **In-cluster consumers:** [Reloader](https://github.com/stakater/Reloader)
  restarts Deployments when the Secret changes.
- **Push to external secret stores** (Vault, AWS SM, GCP SM, Doppler, …):
  create an ESO `PushSecret` referencing our output Secret.
- **Webhook delivery:** ESO `SecretStore` with `provider: webhook` + a
  `PushSecret`. Example YAML in the docs.
- **Cross-tool notification fan-out:** `kubernetes-event-exporter` /
  Argo Events / Alertmanager, consuming the Events and metrics from §1.

**Explicitly not doing:** native webhook field on the CR, direct cloud
secret-manager writes, ConfigMap export, file/log export.

---

## 4. Operator credential management: self-referential CRDs

**Decision:** Support source-specific self-rotate CRDs where the source API
allows it (starting with `GitLabPersonalAccessToken`). For sources without
self-rotate, the root credential stays user-managed.

The existing `apiTokenSecretRef` field is already flexible enough to enable
composition (one CR's output Secret is another CR's API-credential input) —
no new schema required for that case.

### Self-rotate CRD behavior

- New CRD per source: e.g. `GitLabPersonalAccessToken`. Exports to the Secret
  that other CRs reference via `apiTokenSecretRef`.
- First reconcile: adopt the user-created Secret (ownership transfer), call
  the source's self-rotate endpoint, write the new value back.
- Subsequent reconciles: scheduled rotation via the same endpoint.
- Adoption is gated on explicit opt-in: `spec.adoptExistingSecret: true`. If
  unset and the Secret exists, `Ready=False, reason=SecretNotAdopted`.
- On first adoption, emit Event
  `TookOwnership secret=<name>; future rotations will mutate /data/<key> in
  place; ensure your bootstrap tool does not revert this field`.

### Cycle detection (for the composition pattern)

- On reconcile, walk owner-refs from `apiTokenSecretRef` to detect cycles
  among operator-managed Secrets. Bound the walk.
- On cycle: `Ready=False, reason=DependencyCycle` on all CRs in the cycle.
  Don't silently break it.
- Ordering between dependent CRs: rely on natural requeue-on-error
  (NotFound → requeue → retry after dependency mints). No DAG scheduler
  needed.

### Bootstrap paths (documented, not code)

The initial credential value has to come from somewhere. Three supported
paths:

1. **Manual `kubectl create secret`** — lowest friction.
2. **GitOps (SealedSecret / SOPS + ArgoCD/Flux)** — encrypted in git, decrypted
   into a Secret. Requires `ignoreDifferences` on `/data/<key>` in the ArgoCD
   Application (or equivalent) so the sync controller doesn't revert rotated
   values. Example in docs.
3. **ESO `ExternalSecret` with `refreshInterval: 0`** — credential lives in a
   vault, ESO materializes it once, operator takes over. Cleanest separation;
   requires ESO + a vault.

**Key invariant to call out in docs:** the operator writes `/data/<keyName>`;
whatever produces the bootstrap value must not fight the operator over that
field after adoption.

### Explicitly not doing

- Dedicated `Provider` / `TokenProvider` CRD (ESO-style). Deferred per
  CLAUDE.md; still true.
- OIDC / workload-identity federation for source APIs. Most sources we target
  don't support it at the required scopes.

---

## Implementation order (rough)

1. **OTel + Prometheus metrics + Events** (§1). Self-contained, unlocks
   observability for everything else. Extends existing reconciler.
2. **`KeepOld` grace period** (§2). Requires status additions, revoke API
   surface in `internal/sources/gitlab`, reconcile-time validation.
3. **Documentation pass** (§3 and §4 composition patterns). Docs-only; no
   code.
4. **`GitLabPersonalAccessToken` CRD** (§4). New CRD kind; reuses shared
   `TokenSpecBase`. First use of adoption + self-rotate patterns. Needs cycle
   detection in the shared reconcile path.

README's "Open design questions" section gets replaced with a short pointer
to this doc once implementations land.
