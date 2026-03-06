# mapserver-operator
_Kubernetes controller/operator to serve WFS and WMS instances._

[![Build](https://github.com/PDOK/mapserver-operator/actions/workflows/build-and-publish-image.yml/badge.svg)](https://github.com/PDOK/mapserver-operator/actions/workflows/build-and-publish-image.yml)
[![Lint (go)](https://github.com/PDOK/mapserver-operator/actions/workflows/lint.yml/badge.svg)](https://github.com/PDOK/mapserver-operator/actions/workflows/lint.yml)
[![GitHub license](https://img.shields.io/github/license/PDOK/mapserver-operator)](https://github.com/PDOK/mapserver-operator/blob/master/LICENSE)

## Description
This Kubernetes controller cq operator (an operator could be described as a specialized controller)
ensures that the necessary resources are created or kept up-to-date in a cluster
to deploy instances of the [Web Map Service](https://www.ogc.org/standards/wms/)(WMS) and [Web Features Service](https://www.ogc.org/standards/wfs/)(WFS). This repository is a complete solution to deploy WMS and WFS services according to CR schemas.
This operator uses two Custom Resources(CR) called _WMS_ and _WFS_ as the input for the deployment, which is also defined in this repository.

## Getting Started

### Prerequisites
- go version v1.25.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/mapserver-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/mapserver-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Develop

The project is written in Go and scaffolded with [kubebuilder](https://kubebuilder.io).

### kubebuilder

Read the manual when you want/need to make changes.
E.g. run `make test` before committing.

To update the scaffolding:
- Install the latest version of kubebuilder globally on your machine;
- Run: `kubebuilder alpha update --from-branch master`

### Linting

Install [golangci-lint](https://golangci-lint.run/usage/install/) and run `golangci-lint run`
from the root.
(Don't run `make lint`, it uses an old version of golangci-lint.)

# Contributing

### How to contribute
Mapserver-operator is solely developed by PDOK. Contributions are however always welcome. If you have any questions or suggestions you can create an issue in the issue tracker.

### Contact
The maintainers can be contacted through the issue tracker.

# Authors
This project is developed by [PDOK](https://www.pdok.nl/), a platform for publication of geographic datasets of Dutch governmental institutions.
