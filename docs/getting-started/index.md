# Getting Started

Install the operator, apply a sample custom resource, and watch it mint a token.

## Prerequisites

- Go 1.25+
- Docker (or another OCI builder — set `CONTAINER_TOOL` in the Makefile)
- `kubectl` pointed at a Kubernetes cluster
- [kubebuilder](https://book.kubebuilder.io/) (only needed for regenerating
  scaffolded code)

## Walkthrough

1. **[Install](install.md)** — `make install` / `make run` against a cluster.
2. **[Deploy](deploy.md)** — build and push the manager image, `make deploy`.
3. **[Try it](try-it.md)** — create an API-credential Secret, apply a sample
   CR, observe rotation.
4. **[Tear down](tear-down.md)** — remove the CR, uninstall the operator.
