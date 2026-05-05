# Design

The architectural decisions behind Token Rotator. Each page covers one
pillar.

- **[One CRD per source](crd-per-source.md)** — why we rejected a single
  generic `Token` CRD.
- **[Naming](naming.md)** — API group and kind conventions.
- **[Aggregation](aggregation.md)** — shared categories and printer columns.
- **[Shared spec and status](shared-spec.md)** — common fields embedded in
  every source CRD.
- **[Resolutions to open questions](open-questions-resolutions.md)** —
  decisions on failure handling, rotation strategy, export targets, and
  credential management.
