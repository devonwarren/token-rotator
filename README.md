# Token Rotator

> Work in progress. Contributions welcome.

A Kubernetes operator that rotates API tokens for third-party services, modeled
on cert-manager. Each token source (GitLab, Tailscale, Docker Registry, DataDog,
ArgoCD, Google Workspace, …) gets its own strongly-typed CRD, and the operator
mints replacement tokens on a cron schedule or on demand, publishing them to a
Kubernetes Secret.

## Status

| Source | CRD kind | Status |
|---|---|---|
| GitLab — project access tokens | `GitLabProjectAccessToken` | Implemented (v1alpha1) |
| GitLab — group access tokens | `GitLabGroupAccessToken` | Planned |
| GitLab — personal access tokens | `GitLabPersonalAccessToken` | Planned |
| Tailscale | `TailscaleAuthKey` | Planned |
| Docker Hub | `DockerHubAccessToken` | Planned |
| DataDog | `DatadogApplicationKey` | Planned |
| ArgoCD | `ArgoCDProjectToken` | Planned |
| Google Workspace | `GoogleServiceAccountKey` | Planned |

## Design

### One CRD per token type

Rather than a single generic `Token` CRD with an opaque `config` field, each
token source gets its own strongly-typed CRD. The APIs for minting tokens vary
too much between providers — GitLab takes a flat `scopes[]` + integer
`access_level`, Tailscale takes a nested `capabilities` object with tags,
DataDog API keys have no scopes at all, ArgoCD tokens defer permissions to
out-of-band RBAC, etc. — for a single schema to be useful. Per-source CRDs give
us `kubectl apply`-time validation, per-source RBAC, and let each source
version independently.

### Naming

All CRDs live in a single API group: **`token-rotator.org`**. Kinds are
prefixed with the source name to keep them unambiguous (e.g.
`GitLabProjectAccessToken`, `TailscaleAuthKey`). This follows the
cert-manager / external-secrets / ArgoCD convention.

### Aggregation

Every CRD registers into shared categories so all token types can be listed
together:

- `kubectl get tokens` — every rotated token in the cluster, across sources
- `kubectl get gitlab` — every GitLab-sourced token

All token CRDs expose the same baseline printer columns (`Ready`, `Last
Rotated`, `Next Rotation`, `Export`, `Age`) so the aggregated view is usable
without `-o wide`. Source-specific columns (e.g. GitLab `access_level`) are
registered at `priority: 1` and only appear with `-o wide`.

### Shared spec and status

All token CRDs embed a common base spec and status; per-source specs add their
own fields:

```yaml
spec:
  rotationSchedule: "0 0 1 * *"    # cron
  forceNow: false
  rotationStrategy: Immediate       # Immediate | KeepOld
  apiTokenSecretRef:                # the credential the controller uses to mint
    name: gitlab-api-credentials
    key: token
  export:
    type: Secret
    name: gitlab-ci-token
    namespace: default
  # source-specific fields:
  project: mygroup/myproject
  accessLevel: Maintainer
  scopes: [api, read_registry]

status:
  conditions: [...]
  lastRotationTime: ...
  nextRotationTime: ...
  currentTokenRef: {name: ..., namespace: ...}
```

## Getting started

### Prerequisites

- Go 1.25+
- Docker
- `kubectl` pointed at a Kubernetes cluster
- [kubebuilder](https://book.kubebuilder.io/) for regenerating scaffolded code

### Build and run locally against a cluster

Install the CRDs into the currently-configured cluster:

```sh
make install
```

Run the controller against that cluster from your workstation (outside the
cluster, using your kubeconfig):

```sh
make run
```

### Deploy to the cluster

Build and push the image, then deploy the manager:

```sh
make docker-build docker-push IMG=ghcr.io/devonwarren/token-rotator:dev
make deploy IMG=ghcr.io/devonwarren/token-rotator:dev
```

### Try it

Create a Secret holding the GitLab API token the operator will use to mint
project access tokens, then apply the sample CR:

```sh
kubectl create secret generic gitlab-api-credentials --from-literal=token=<your-gitlab-pat>
kubectl apply -k config/samples/
kubectl get tokens
```

### Tear down

```sh
kubectl delete -k config/samples/
make undeploy
make uninstall
```

## Repo layout

```
api/v1alpha1/       # CRD types (generated OpenAPI schemas in config/crd/bases/)
cmd/main.go         # manager entrypoint
internal/
  controller/       # per-CRD reconcilers
  rotation/         # shared helpers: schedule, conditions, export
  sources/          # per-source API clients (gitlab, …)
config/             # kustomize manifests (CRDs, RBAC, deployment)
```

## Open design questions

- How to handle rotation failures — metrics-only, Argo Notifications, or
  status-only?
- `KeepOld` rotation strategy — how long to retain the previous token before
  revoking it?
- Additional export targets beyond Kubernetes Secret — PushSecret, webhook?
- Managing the controller's own API credentials — self-referential CRD?

## License

Apache 2.0. See individual source files for copyright.
