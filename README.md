* Source
  * Gitlab Token
  * Tailscale
  * Docker Registry
  * Datadog (low)
  * Google Group (low)
  * ArgoCD (low)
* Required permissions
* Last rotation timestamp
* Expiration timestamp
* Additional params
  * Ex: gitlab URL,
* Rotation Schedule
  * CRON
  * Force Now (rotate with set to true, then update to the value of false when done)
* Rotation Strategy
  * Keep Old Secret Active
    * How Long?
  * Immediate
* Secret Export ideas
  * PushSecret
  * Secret
  * Direct resource yaml change (ytt, jsonpatch?)