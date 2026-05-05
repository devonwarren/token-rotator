# GitLab project access token

Rotates a GitLab [project access token](https://docs.gitlab.com/ee/user/project/settings/project_access_tokens.html)
on a cron schedule.

**Kind:** `GitLabProjectAccessToken`
**API group:** `token-rotator.org`
**Versions:** `v1alpha1`
**Status:** implemented.

## API credential

The CR references a Kubernetes Secret via `spec.apiTokenSecretRef`. That Secret
must hold a GitLab token with permission to create and revoke project access
tokens on the target project — typically a maintainer-or-higher personal
access token, group access token, or a bootstrapping project access token.

Required scopes: `api`.

!!! tip "Rotating the bootstrap credential itself"
    See [Guides › Bootstrap credentials](../guides/bootstrap-credentials.md).

## Example

```yaml
apiVersion: token-rotator.org/v1alpha1
kind: GitLabProjectAccessToken
metadata:
  name: ci-token
spec:
  project: mygroup/myproject
  accessLevel: Maintainer
  scopes: [api, read_registry]
  rotationSchedule: "0 0 1 * *"    # monthly, midnight UTC
  rotationStrategy: Immediate
  apiTokenSecretRef:
    name: gitlab-api-credentials
    key: token
  export:
    type: Secret
    name: gitlab-ci-token
```

## Status

- `status.lastRotationTime` — when the last successful rotation completed.
- `status.nextRotationTime` — when the next rotation will occur.
- `status.currentTokenRef` — the Secret holding the current token value.
- `status.conditions` — `Ready`, with reason/message on failure.

## Gotchas

- **Lifetime.** The minted token's expiry is set to `2 × rotationInterval`,
  giving a missed reconcile a second chance before consumers break.
- **Access level mapping.** `accessLevel: Maintainer` maps to GitLab's
  integer `40`. See the GitLab API reference for the full mapping.
- **Revocation is idempotent.** A 404 from GitLab on revoke is treated as
  success (previously revoked or never existed).
