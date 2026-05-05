# Deploy

Build and push the image, then deploy the manager:

```sh
make docker-build docker-push IMG=ghcr.io/devonwarren/token-rotator:dev
make deploy IMG=ghcr.io/devonwarren/token-rotator:dev
```

The manager Deployment is configured with a hardened security context
(`runAsNonRoot`, `readOnlyRootFilesystem`, `allowPrivilegeEscalation: false`,
all capabilities dropped, `seccompProfile: RuntimeDefault`). See
`config/manager/manager.yaml`.
