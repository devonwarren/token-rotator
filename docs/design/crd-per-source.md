# One CRD per token source

Rather than a single generic `Token` CRD with an opaque `config` field, each
token source gets its own strongly-typed CRD. The APIs for minting tokens vary
too much between providers — GitLab takes a flat `scopes[]` + integer
`access_level`, Tailscale takes a nested `capabilities` object with tags,
DataDog API keys have no scopes at all, ArgoCD tokens defer permissions to
out-of-band RBAC, etc. — for a single schema to be useful.

Per-source CRDs give us:

- **`kubectl apply`-time validation** via OpenAPI schema per source.
- **Per-source RBAC** — grant minting rights for GitLab tokens without
  granting anything on Tailscale keys.
- **Independent versioning** — a source can move from `v1alpha1` to `v1beta1`
  without dragging the others.

A generic `CustomToken` escape-hatch CRD may be added later but is not the
default.
