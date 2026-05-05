# Guides

Task-oriented walkthroughs for common integration patterns.

- **[Bootstrap credentials](bootstrap-credentials.md)** — manual, GitOps, and
  ESO paths for getting the operator's initial API credential into the
  cluster.
- **[ArgoCD integration](argocd-integration.md)** — how to stop ArgoCD from
  fighting the operator over rotated Secret values.
- **[Pod reload with Reloader](pod-reload-with-reloader.md)** — restart
  consumer Pods automatically when a rotated Secret changes.
- **[ESO PushSecret composition](eso-pushsecret-composition.md)** — deliver
  rotated tokens to external stores or webhook endpoints via
  external-secrets.
- **[Observability](observability.md)** — Kubernetes Events, Prometheus
  metrics, and OTLP export.
