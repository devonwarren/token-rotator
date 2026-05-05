# Events

!!! info "Planned"
    Event reasons below are the committed design; implementation is in
    progress.

All events are emitted on the CR object — `kubectl describe <kind> <name>`
shows them, and `kubectl get events --field-selector involvedObject.name=<name>`
lists them in time order.

| Reason | Type | When |
|---|---|---|
| `RotationStarted` | Normal | A rotation reconcile has begun. |
| `RotationSucceeded` | Normal | New token minted and exported successfully. |
| `RotationFailed` | Warning | Rotation failed; message contains a non-sensitive reason. |
| `TokenRevoked` | Normal | A previous token has been revoked (end of `KeepOld` grace period, or `Immediate` replacement). |
| `ExportUpdated` | Normal | The exported Secret's value has been updated. |
| `TookOwnership` | Normal | The controller adopted a pre-existing Secret (self-rotate CRDs). |
| `DependencyCycle` | Warning | Two or more operator-managed Secrets reference each other via `apiTokenSecretRef`. |
| `SecretNotAdopted` | Warning | A self-rotate CRD references a pre-existing Secret without `adoptExistingSecret: true`. |
| `InvalidGracePeriod` | Warning | `KeepOld.gracePeriod` is >= `rotationInterval`; would produce two valid tokens indefinitely. |

## Never emitted

Event messages never contain token values or API credentials.
