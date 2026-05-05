# Tear down

```sh
kubectl delete -k config/samples/
make undeploy
make uninstall
```

`make undeploy` removes the manager; `make uninstall` removes the CRDs. Order
matters — deleting CRDs first would strand finalizers.
