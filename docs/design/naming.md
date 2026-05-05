# Naming

All CRDs live in a single API group: **`token-rotator.org`**. Kinds are
prefixed with the source name to keep them unambiguous (e.g.
`GitLabProjectAccessToken`, `TailscaleAuthKey`). This follows the
cert-manager / external-secrets / ArgoCD convention.

If the project ever grows to Crossplane scale (dozens of kinds per source) the
trade-off to per-source API subgroups could be revisited — but we're not close.
