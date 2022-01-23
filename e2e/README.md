# How to run E2E tests in local
## Setup kind

Install kind, please refer: https://kind.sigs.k8s.io/

And bootstrap a cluster.

```
$ cat kind.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
  - role: worker
  - role: worker
$ kind create cluster --config ./kind.yaml --kubeconfig ~/.kube/config-kind
$ export KUBECONFIG=~/.kube/config-kind
```

Then install cert-manager.
```
$ kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.6.1/cert-manager.yaml
```

## Build docker image and push

Build docker image.

```
$ cd fluentd-sidecar-injector
$ docker build -t my-docker-registry/fluentd-sidecar-injector:experimental .
$ docker push my-docker-registry/fluentd-sidecar-injector:experimental
```

## Install ginkgo

```
$ go get github.com/onsi/ginkgo/ginkgo
```

## Run E2E tests

```
$ export FLUENTD_SIDECAR_INJECTOR_IMAGE=my-docker-registry/fluentd-sidecar-injector:experimental
$ ginkgo -r ./e2e
```
