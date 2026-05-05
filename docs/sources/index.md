# Sources

Each token source is a separate CRD under the `token-rotator.org` API group.

| Source | CRD kind | Status | Docs |
|---|---|---|---|
| GitLab — project access tokens | `GitLabProjectAccessToken` | Implemented (v1alpha1) | [Reference](gitlab-project-access-token.md) |
| GitLab — group access tokens | `GitLabGroupAccessToken` | Planned | — |
| GitLab — personal access tokens | `GitLabPersonalAccessToken` | Planned | — |
| Tailscale | `TailscaleAuthKey` | Planned | — |
| Docker Hub | `DockerHubAccessToken` | Planned | — |
| DataDog | `DatadogApplicationKey` | Planned | — |
| ArgoCD | `ArgoCDProjectToken` | Planned | — |
| Google Workspace | `GoogleServiceAccountKey` | Planned | — |

Each source page covers:

- The CR schema (spec/status fields).
- Required scopes on the API credential.
- A minimal example CR.
- Source-specific gotchas (rate limits, revocation semantics, etc.).
