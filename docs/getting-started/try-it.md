# Try it

Create a Secret holding the GitLab API token the operator will use to mint
project access tokens, then apply the sample CR:

```sh
kubectl create secret generic gitlab-api-credentials \
  --from-literal=token=<your-gitlab-pat>

kubectl apply -k config/samples/

kubectl get tokens
```

`kubectl get tokens` aggregates across every token source — see
[Design › Aggregation](../design/aggregation.md).

For per-source details, see
[Sources › GitLab project access token](../sources/gitlab-project-access-token.md).
