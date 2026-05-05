# Bootstrap credentials

The operator needs an API credential to mint tokens. That credential has to
exist in the cluster before the first reconcile — you can't materialize a
credential from nothing.

After bootstrap, the operator can manage its own credential rotation for
sources that support self-rotate (e.g. GitLab PATs). This page walks through
each bootstrap option.

## Three paths

=== "Manual"

    Lowest friction; fine for single-cluster or evaluation setups.

    ```sh
    kubectl create secret generic gitlab-api-credentials \
      --from-literal=token=<your-gitlab-pat>
    ```

    Reference the Secret from your CR's `spec.apiTokenSecretRef`.

=== "GitOps (SealedSecret / SOPS)"

    Encrypt the credential in git, let ArgoCD/Flux decrypt it into a
    Secret. See [ArgoCD integration](argocd-integration.md) for the
    `ignoreDifferences` pattern you need once the operator starts rotating
    the value.

=== "ESO (External Secrets Operator)"

    Keeps the bootstrap credential out of the cluster entirely. Credential
    lives in a vault (Vault, AWS Secrets Manager, Doppler, …); ESO
    materializes it once and the operator takes over.

    ```yaml
    apiVersion: external-secrets.io/v1beta1
    kind: ExternalSecret
    metadata:
      name: gitlab-api-credentials
    spec:
      refreshInterval: "0"      # fetch once, then never again
      secretStoreRef:
        name: vault
        kind: ClusterSecretStore
      target:
        name: gitlab-api-credentials
        creationPolicy: Owner
      data:
        - secretKey: token
          remoteRef:
            key: secret/gitlab
            property: bootstrap-pat
    ```

## Self-rotating the bootstrap credential (planned)

For sources that support it, a dedicated CRD (e.g.
`GitLabPersonalAccessToken`) will rotate the bootstrap credential in place,
so after first install there is no long-lived static credential to manage.

!!! info "Planned"
    Not yet implemented. See
    [Resolutions to open questions §4](../design/open-questions-resolutions.md#4-operator-credential-management-self-referential-crds).

## Invariant to remember

After the operator adopts a Secret, it writes `/data/<keyName>` on every
rotation. Whatever produces the bootstrap value must not fight the operator
over that field — see [ArgoCD integration](argocd-integration.md) for the
canonical example.
