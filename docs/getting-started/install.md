# Install

Install the CRDs into the currently-configured cluster:

```sh
make install
```

Run the controller against that cluster from your workstation (outside the
cluster, using your kubeconfig):

```sh
make run
```

This is the quickest path for local iteration. To deploy the controller *into*
the cluster, see [Deploy](deploy.md).
