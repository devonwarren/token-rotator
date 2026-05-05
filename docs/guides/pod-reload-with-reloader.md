# Pod reload with Reloader

The operator's contract ends at "Secret contains current value." How
consumers pick up the new value is up to them.

Kubelet refreshes mounted Secrets automatically (with some delay), but env-var
Secrets and in-memory caches don't. [Stakater
Reloader](https://github.com/stakater/Reloader) watches Secret changes and
triggers rolling restarts of Deployments/StatefulSets/DaemonSets that reference
them.

## Annotation on the consumer

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  annotations:
    secret.reloader.stakater.com/reload: "gitlab-ci-token"
spec:
  template:
    spec:
      containers:
        - name: app
          envFrom:
            - secretRef:
                name: gitlab-ci-token
```

When the operator rotates `gitlab-ci-token`, Reloader triggers a rolling
restart of `my-app`.

## Pairing with `KeepOld`

If consumer restart is slow (large images, long readiness probes), use the
`KeepOld` rotation strategy with a grace period long enough to cover the
restart:

```yaml
spec:
  rotationStrategy:
    type: KeepOld
    keepOld:
      gracePeriod: 1h
```

The previous token stays valid for `gracePeriod`, which bounds the window
where a pod on the old Secret is briefly running alongside pods on the new
Secret. See [Resolutions to open questions
§2](../design/open-questions-resolutions.md#2-keepold-grace-period).
