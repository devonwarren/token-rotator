# Aggregation

Every CRD registers into shared categories so all token types can be listed
together:

- `kubectl get tokens` — every rotated token in the cluster, across sources.
- `kubectl get gitlab` — every GitLab-sourced token.

All token CRDs expose the same baseline printer columns — `Ready`, `Last
Rotated`, `Next Rotation`, `Export`, `Age` — so the aggregated view is usable
without `-o wide`.

Source-specific columns (e.g. GitLab `access_level`) are registered at
`priority: 1` and only appear with `-o wide`.
