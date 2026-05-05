# ESO PushSecret composition

The operator only exports rotated tokens to Kubernetes `Secret`s. For
anything else — external vaults, webhook endpoints, multi-cluster sync — the
supported pattern is [External Secrets Operator `PushSecret`](https://external-secrets.io/latest/api/pushsecret/),
which references the Secret the operator writes and pushes it to any
configured `SecretStore`.

## Push to an external vault

```yaml
apiVersion: external-secrets.io/v1beta1
kind: PushSecret
metadata:
  name: gitlab-ci-token-to-vault
spec:
  selector:
    secret:
      name: gitlab-ci-token       # the Secret our operator writes
  secretStoreRefs:
    - name: vault
      kind: ClusterSecretStore
  data:
    - match:
        secretKey: token
        remoteRef:
          remoteKey: secret/gitlab/ci-token
```

## Push to a webhook

ESO has a webhook provider. Configure a `SecretStore` with `provider:
webhook`, point a `PushSecret` at it, and ESO POSTs the secret value on
change.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: rotation-webhook
spec:
  provider:
    webhook:
      url: "https://my-endpoint.example.com/rotated"
      method: POST
      headers:
        Content-Type: application/json
      body: |
        {"token": "{{ .token }}"}
---
apiVersion: external-secrets.io/v1beta1
kind: PushSecret
metadata:
  name: gitlab-ci-token-webhook
spec:
  selector:
    secret:
      name: gitlab-ci-token
  secretStoreRefs:
    - name: rotation-webhook
  data:
    - match:
        secretKey: token
        remoteRef:
          remoteKey: token
```

## Why not a native webhook field on the CR?

ESO's webhook provider solves the problem — auth, TLS, body templating,
retries — better than we would. See [Resolutions to open questions
§3](../design/open-questions-resolutions.md#3-export-targets).
