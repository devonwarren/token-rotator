# Metrics

!!! info "Planned"
    Metric names below are the committed design; implementation is in
    progress.

All metrics are exported via the OpenTelemetry SDK to both a Prometheus
exporter (controller-runtime's `/metrics` endpoint) and optionally OTLP.

## Inventory

| Name | Type | Labels | Description |
|---|---|---|---|
| `token_rotator_rotation_attempts_total` | counter | `kind, source, namespace, name, result` | Rotations attempted. `result` ∈ `success|failure`. |
| `token_rotator_rotation_failures_total` | counter | `kind, source, namespace, name, reason` | Rotation failures, labelled by failure reason. |
| `token_rotator_last_success_timestamp_seconds` | gauge (async) | `kind, source, namespace, name` | Unix seconds of last successful rotation. |
| `token_rotator_token_expiry_timestamp_seconds` | gauge (async) | `kind, source, namespace, name` | Unix seconds when the current token expires. |

## Cardinality

Labels include per-CR identity (`namespace`, `name`). Follows the
cert-manager convention. If you have thousands of CRs, keep this in mind
when sizing Prometheus storage.

## Never emitted

Metric labels never contain token values or API credentials. If you find one,
that's a bug — report it.
