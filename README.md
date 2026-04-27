# Token Rotator

> This is very much still a work-in-progress but contributions are always welcome!

## Summary

The idea behind token rotator is to automatically manage token rotation for various
products using Kubernetes similar to cert-manager. Sources (ex: GitLab, Tailscale,
Docker Registry, etc) are plugins that define what permissions they support and how
to generate new tokens. The rotation app then mints replacement tokens on-demand or
via cron and exports them using a number of options.

## Source Ideas

- GitLab Access Tokens (personal, project, group)
- Tailscale Auth Keys
- Docker Registry
- DataDog
- ArgoCD
- Google Group

## Design

### One CRD per token type

Rather than a single generic `Token` CRD with an opaque `config` field, each token
source gets its own strongly-typed CRD. The APIs for minting tokens vary too much
between providers — GitLab takes a flat `scopes[]` + integer `access_level`, Tailscale
takes a nested `capabilities` object with tags, DataDog API keys have no scopes at all,
ArgoCD tokens defer permissions to out-of-band RBAC, etc. — for a single schema to be
useful. Per-source CRDs give us `kubectl apply`-time validation, per-source RBAC, and
let each source version independently.

A generic `CustomToken` CRD with a webhook export may be added later as an escape hatch
for sources that don't have first-class support yet.

### Naming

All CRDs live in a single API group: **`token-rotator.org`**. Kinds are prefixed with
the source name to keep them unambiguous:

| Kind | Plural |
|------|--------|
| `GitLabProjectAccessToken` | `gitlabprojectaccesstokens` |
| `GitLabGroupAccessToken` | `gitlabgroupaccesstokens` |
| `GitLabPersonalAccessToken` | `gitlabpersonalaccesstokens` |
| `TailscaleAuthKey` | `tailscaleauthkeys` |
| `DockerHubAccessToken` | `dockerhubaccesstokens` |
| `DatadogApplicationKey` | `datadogapplicationkeys` |
| `ArgoCDProjectToken` | `argocdprojecttokens` |
| `GoogleServiceAccountKey` | `googleserviceaccountkeys` |

This follows the cert-manager / ESO / ArgoCD convention (one group, several kinds).
If the project ever grows to Crossplane scale — dozens of kinds per source — we'd
revisit moving to per-source subgroups (e.g. `gitlab.token-rotator.org`).

### Aggregation

Every CRD registers into shared categories so all token types can be listed together:

- `kubectl get tokens` — every rotated token in the cluster, across sources
- `kubectl get gitlab` — every GitLab-sourced token

Every kind also exposes the same baseline printer columns (`READY`, `LAST ROTATED`,
`NEXT ROTATION`, `EXPORT`, `AGE`) so the aggregated view is usable without `-o wide`.
Source-specific columns (e.g. GitLab `access_level`, Tailscale `tags`) are registered
at `priority: 1` so they only appear with `-o wide`.

### Shared spec and status

All token CRDs share a common base spec and status; subclasses add source-specific
fields under their own section of `spec`:

```yaml
spec:
  rotationSchedule: "0 0 1 * *"   # cron
  forceNow: false
  rotationStrategy: Immediate      # Immediate | KeepOld
  export:
    type: Secret
    name: gitlab-ci-token
    namespace: ci
  # source-specific fields here, e.g. for GitLabProjectAccessToken:
  project: healthtensor/iac
  accessLevel: Maintainer
  scopes: [api, read_registry]

status:
  conditions: [...]
  lastRotationTime: ...
  nextRotationTime: ...
  currentTokenRef: {name: ..., namespace: ...}
  previousTokenRef: {name: ..., namespace: ...}  # only for KeepOld
```

## TBD Design Decisions

- How to input additional per-source parameters (e.g. self-hosted GitLab URLs)
- How to handle failures
  - Rely on metrics export and Prometheus alarms
  - Implement Argo Notifications
  - Failure status on token object
- Rotation strategy implementation — should `KeepOld` archive the previous token,
  and for how long, before invalidating it?
- How to export the new token
  - Kubernetes Secret objects (could use [PushSecret](https://external-secrets.io/latest/api/pushsecret/) to manage afterwards)
  - A [Fake External Secrets Operator](https://external-secrets.io/latest/provider/fake/)
  - Custom Webhook (would require the user to do more dev work)
- Managing the Token Rotator's API access token itself
  - A separate CRD just for itself? Set it to autorotate — could be good from an RBAC perspective
