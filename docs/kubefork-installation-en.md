# How to deploy kubefork

This document describes how to deploy kubefork in a cluster.

- [How to deploy kubefork](#how-to-deploy-kubefork)
  - [Requirement](#requirement)
  - [Introduction of kubefork-controller](#introduction-of-kubefork-controller)
  - [Introduction of ForkManager](#introduction-of-forkmanager)
  - [Installation of kubeforkctl](#installation-of-kubeforkctl)
  - [References](#references)

---

## Requirement

Go 1.16+
kubernetes 1.25+

In addition, please install the following into your cluster in advance
[deployment-duplicator](https://github.com/wantedly/deployment-duplicator)
[Istio 1.15+](https://istio.io/latest/docs/setup/getting-started/)
[Emissary-ingress 3.1+](https://www.getambassador.io/docs/emissary/latest/tutorials/getting-started/)

## Introduction of kubefork-controller

See [kubefork-controller](https://github.com/wantedly/kubefork-controller).

## Introduction of ForkManager

In order to perform kubefork, it is necessary to apply the ForkManager resource, which configures which headers are looked at and which hosts and services correspond to. The most basic ForkManager manifest is as follows

```yaml
apiVersion: fork.k8s.wantedly.com/v1beta1
kind: ForkManager
metadata:
  name: kubefork-manager
  namespace: manager
spec:
  ambassadorID: fork-ambassador
  headerKey: x-fork-identifier
  upstreams:
  - host: example.com
    original: your-service:80
```

In this case, the header key referenced to distribute communications is `x-fork-identifier`, and communications to `example.com` are normally sent to port 80 of the service `your-service`.

For a detailed specification of ForkManager, see [forkmanager_types.go](https://github.com/wantedly/kubefork-controller/blob/main/api/v1beta1/forkmanager_types.go).

## Installation of kubeforkctl

After applying ForkManager, you should install the command line tool [kubeforkctl](https://github.com/wantedly/kubefork-controller/tree/main/kubeforkctl).

## References

Currently, in order to do kubefork, **users must have the ability to propagate certain headers in their applications**. See [the kubefork documentation](./kubefork-doc-en.md#how-to-do-kubefork) for details.
