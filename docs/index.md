# Token Rotator

A Kubernetes operator that rotates API tokens for third-party services, modeled
on cert-manager. Each token source (GitLab, Tailscale, Docker Registry, DataDog,
ArgoCD, Google Workspace, …) gets its own strongly-typed CRD, and the operator
mints replacement tokens on a cron schedule or on demand, publishing them to a
Kubernetes Secret.

!!! warning "Work in progress"
    This project is under active development. APIs are `v1alpha1` and may
    change. Contributions welcome.

## Source status

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

## Where to go next

- **[Getting Started](getting-started/index.md)** — install the operator and
  rotate your first token.
- **[Design](design/index.md)** — the architectural decisions behind the
  project.
- **[Sources](sources/index.md)** — per-source reference and examples.
- **[Guides](guides/index.md)** — bootstrap patterns, ArgoCD integration,
  Reloader, ESO composition, observability.
- **[Reference](reference/index.md)** — CRD API, metrics, events.
