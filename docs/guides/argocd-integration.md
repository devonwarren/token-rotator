# ArgoCD integration

If you deploy rotated Secrets from git via ArgoCD, auto-sync will fight the
operator: it reverts the rotated value back to what's in git. Tell ArgoCD to
ignore the specific data key the operator writes.

## `ignoreDifferences`

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
spec:
  source: { ... }
  destination: { ... }
  syncPolicy:
    automated:
      selfHeal: true
      prune: true
  ignoreDifferences:
    - group: ""
      kind: Secret
      name: gitlab-api-credentials
      jsonPointers:
        - /data/token
```

With this in place, ArgoCD still creates the Secret from your git manifest on
first sync, and the operator takes ownership of `/data/token` on first
reconcile. Subsequent rotations don't register as drift.

## When to use which

- **ArgoCD-managed Secret, operator rotates in place:** use
  `ignoreDifferences` as above.
- **ESO-managed Secret:** set `refreshInterval: "0"` on the `ExternalSecret`
  so ESO doesn't refresh a field the operator now owns.
- **Manual `kubectl create secret`:** nothing to do.

## Adoption signal

On first adoption, the operator emits an Event:

> `TookOwnership secret=gitlab-api-credentials; future rotations will mutate /data/token in place; ensure your bootstrap tool does not revert this field`

Watch for it with `kubectl get events --field-selector reason=TookOwnership`.
