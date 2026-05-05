# Shared spec and status

All token CRDs embed a common base spec and status; per-source specs add their
own fields on top.

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

The shared types live in `api/v1alpha1/tokenbase_types.go`. Shared behavior
(schedule parsing, condition helpers, export) lives in
`internal/rotation/`.

No `TokenSource` interface: the specs vary too much to abstract usefully. Each
per-source reconciler calls its concrete client directly.
