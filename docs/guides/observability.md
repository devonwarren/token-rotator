# Observability

The operator exposes three observability surfaces:

- **Kubernetes Events** on each CR for every rotation lifecycle transition.
- **Prometheus-compatible metrics** on the controller-runtime metrics
  endpoint.
- **OTLP metrics export**, configurable via environment variables for users
  on OpenTelemetry-native stacks.

Metrics are instrumented via the OpenTelemetry SDK and exported through
*both* a Prometheus exporter (scrape endpoint) and optionally OTLP (push).

!!! info "Planned"
    Metric names and event reasons below are the committed design; the
    implementation is in progress. See [Resolutions to open questions
    §1](../design/open-questions-resolutions.md#1-failure-handling-observability-surface).

## Events

Emitted on the CR object. Discover with:

```sh
kubectl describe <kind> <name>
kubectl get events --field-selector involvedObject.name=<name>
```

See [Reference › Events](../reference/events.md) for the full list.

## Prometheus metrics

Scrape the controller's `/metrics` endpoint. See [Reference ›
Metrics](../reference/metrics.md) for metric names, labels, and semantics.

Example Alertmanager rule — rotation hasn't succeeded in more than twice the
rotation interval:

```yaml
- alert: TokenRotationStalled
  expr: |
    time() - token_rotator_last_success_timestamp_seconds
      > on(kind,namespace,name) 2 * token_rotator_rotation_interval_seconds
  for: 5m
```

## OTLP export

Set `OTEL_EXPORTER_OTLP_ENDPOINT` (and related OTel env vars) on the manager
Deployment to enable OTLP metrics push. Disabled by default.
