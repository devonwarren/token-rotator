# Token Rotator

> This is very much still a work-in-progress but contributions are always welcome! 

## Summary

The idea behind token rotator is to automatically manage token rotation for various products using Kubernetes similar to cert-manager. I'd like to have the sources (ex: Gitlab, Tailscale, Docker Registry, etc) be plugins that support definitions of what permissions and how to generate the new tokens. Then we'd have the rotation app generate replacement tokens on-demand or via cron, and would then export them using a number of options.

## Source Ideas

- Gitlab Access Tokens
- Tailscale Tokens
- Docker Registry
- DataDog
- ArgoCD
- Google Group

## TBD Design Decisions

- How to handle required permissions sent when generating new tokens
  - Raw JSON - less documented but wouldn't require complicated CRD definitions
  - Structured Pydantic models, more overhead on plugins and may require many CRD types
- How to input additional parameters such as Gitlab Url or project vs group level
- How to handle failures
  - Rely on metrics export and prometheus alarms
  - Implement Argo Notifications
  - Failure status on token object
- Rotation strategy, should we invalidate the original token immediately or leave some time to propogate and then archive
- How do we export the new tokend
  - Kubernetes Secret objects (Could use [PushSecret](https://external-secrets.io/latest/api/pushsecret/) to manage afterwards)
  - A [Fake External Secrets Operator](https://external-secrets.io/latest/provider/fake/)
  - Custom Webhook (would require the user to do more dev work)
- Managing the Token Rotator's API access token itself
  - A separate CRD just for itself? Set it to autorotate, could be good from an RBAC perspective
