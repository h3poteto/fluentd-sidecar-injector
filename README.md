[![CircleCI](https://circleci.com/gh/h3poteto/fluentd-sidecar-injector.svg?style=svg)](https://circleci.com/gh/h3poteto/fluentd-sidecar-injector)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/h3poteto/fluentd-sidecar-injector?sort=semver&style=square)
[![Dependabot](https://img.shields.io/badge/Dependabot-enabled-blue.svg)](https://dependabot.com)

# fluentd-sidecar-injector

`fluentd-sidecar-injector` is a webhook server for kubernetes admission webhook. This server inject fluentd or fluent-bit container as sidecar for specified Pod using mutation webhook. The feature is

- Automatically sidecar injection
- You can control injection using Pod's annotations
- You can change fluentd or fluent-bit docker image to be injected

## Install
You can install this controller and webhook server using helm.

```
$ helm repo add h3poteto-stable https://h3poteto.github.io/charts/stable
$ helm install my-injector --namespace kube-system h3poteto-stable/fluentd-sidecar-injector
```

Please refer [helm repository](https://github.com/h3poteto/charts/tree/master/stable/fluentd-sidecar-injector) for parameters.


After install it, custom resources and controller will be installed.

```
$ kubectl get sidecarinjectors -n kube-system
NAME                  AGE
my-injector-fluentd   1m56s

$ kubectl get pods -n kube-system -l operator.h3poteto.dev=control-plane
NAME                                   READY   STATUS    RESTARTS   AGE
my-injector-manager-6d7f6bcd55-z5jcv   1/1     Running   0          2m17s
```

And it creates admission webhook for the sidecar injector.

```
$ kubectl get mutatingwebhookconfigurations
NAME                                           WEBHOOKS   AGE
sidecar-injector-webhook-my-injector-fluentd   1          5m15s

$ kubectl get svc -n kube-system -l sidecarinjectors.operator.h3poteto.dev=webhook-service
NAME                                   TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
sidecar-injector-my-injector-fluentd   ClusterIP   100.69.147.98   <none>        443/TCP   4m2s

$ kubectl get pods -n kube-system -l sidecarinjectors.operator.h3poteto.dev=webhook-pod
NAME                                           READY   STATUS    RESTARTS   AGE
my-injector-fluentd-handler-5969df9695-ftklp   1/1     Running   0          4m51s
my-injector-fluentd-handler-5969df9695-x5n5r   1/1     Running   0          4m51s
```

## Usage

After you install this webhook server, fluentd sidecar containers are automatically injected, if you specify the annotation `fluentd-sidecar-injector.h3poteto.dev/injection: 'enabled'` to the pods.

For example:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-test
  labels:
    app: nginx-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx-test
  template:
    metadata:
      annotations:
        fluentd-sidecar-injector.h3poteto.dev/injection: 'enabled'
        fluentd-sidecar-injector.h3poteto.dev/application-log-dir: '/var/log/nginx'
        fluentd-sidecar-injector.h3poteto.dev/collector: 'fluentd'
      labels:
        app: nginx-test
    spec:
      containers:
        - name: nginx
          image: nginx:latest
```

FluentD is injected for this Pod.

```sh
$ kubectl get pod
NAME                          READY   STATUS    RESTARTS   AGE
nginx-test-6cbf4485f8-kq8ws   2/2     Running   0          9s
```

```sh
$ kubectl describe pod nginx-test-6cbf4485f8-kq8ws
Name:           nginx-test-6cbf4485f8-kq8ws
Namespace:      default
Containers:
  nginx:
    Container ID:   docker://ce74393381205786668a1fe2a4bc83ba058d380714b8a7ddca23966c8c7f0eb0
    Image:          nginx:latest
    Image ID:       docker-pullable://nginx@sha256:ad5552c786f128e389a0263104ae39f3d3c7895579d45ae716f528185b36bc6f
    Port:           <none>
    Host Port:      <none>
    State:          Running
      Started:      Fri, 14 Feb 2020 13:49:21 +0900
    Ready:          True
    Restart Count:  0
    Environment:    <none>
    Mounts:
      /var/log/nginx from fluentd-sidecar-injector-logs (rw)
      /var/run/secrets/kubernetes.io/serviceaccount from default-token-8rcns (ro)
  fluentd-sidecar:
    Container ID:   docker://49503c3836fa5ebc40c55db3717f16f21fbdbfaae8859a8ed8a366d04a2b6d9b
    Image:          ghcr.io/h3poteto/fluentd-forward:latest
    Image ID:       docker-pullable://ghcr.io/h3poteto/fluentd-forward@sha256:5d93af333ad9fefbfcb8013d20834fd89c2bbd3fe8b9b9bfa620ded29d7b3205
    Port:           <none>
    Host Port:      <none>
    State:          Running
      Started:      Fri, 14 Feb 2020 13:49:23 +0900
    Ready:          True
    Restart Count:  0
    Limits:
      memory:  1000Mi
    Requests:
      cpu:     100m
      memory:  200Mi
    Environment:
      AGGREGATOR_HOST:      127.0.0.1
      APPLICATION_LOG_DIR:  /var/log/nginx
      TAG_PREFIX:           prod
      TIME_KEY:             time
    Mounts:
      /var/log/nginx from fluentd-sidecar-injector-logs (rw)
```

### Custom fluent.conf

If you need to use your own fluent.conf, use config-volume option.
The following yaml has fluent-conf configmap. It will be mounted on `/fluentd/etc/fluent/fluent.conf`.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-test
  labels:
    app: nginx-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx-test
  template:
    metadata:
      annotations:
        fluentd-sidecar-injector.h3poteto.dev/injection: 'enabled'
        fluentd-sidecar-injector.h3poteto.dev/collector: 'fluentd'
        fluentd-sidecar-injector.h3poteto.dev/docker-image: 'fluent/fluentd:latest'
        fluentd-sidecar-injector.h3poteto.dev/application-log-dir: '/var/log/nginx'
        fluentd-sidecar-injector.h3poteto.dev/aggregator-host: 'fluentd.example.com'
        fluentd-sidecar-injector.h3poteto.dev/config-volume: 'fluent-conf'
      labels:
        app: nginx-test
    spec:
      containers:
        - name: nginx
          image: nginx:latest
    volumes:
      - name: fluent-conf
        configMap:
          name: fluent-conf
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluent-conf
  labels:
    app: fluent-conf
data:
  fluent.conf: |-
    <source>
      @type tail
      path "#{ENV['APPLICATION_LOG_DIR']}/*.access.log"
      pos_file /var/tmp/application.log.pos
      tag "app.*"
      <parse>
        @type ltsv
      </parse>
    </source>

    <filter app.*>
      @type record_transformer
      <record>
        hostname "#{Socket.gethostname}"
      </record>
    </filter>

    <match app.*>
      @type forward

      <server>
        host "#{ENV['AGGREGATOR_HOST']}"
        port "#{ENV['AGGREGATOR_PORT']} || 24224"
      </server>
    </match>
```

### Annotations

Please specify these annotations to your pods like [this](example/deployment.yaml).

| Name                                                                              | Required | Default                           |
| --------------------------------------------------------------------------------- | -------- | --------------------------------- |
| [fluentd-sidecar-injector.h3poteto.dev/injection](#injection)                     | optional | ""                                |
| [fluentd-sidecar-injector.h3poteto.dev/docker-image](#docker-image)               | optional | `ghcr.io/h3poteto/fluentd-forward:latest` |
| [fluentd-sidecar-injector.h3poteto.dev/collector](#collector)                     | optional | `fluentd`                         |
| [fluentd-sidecar-injector.h3poteto.dev/aggregator-host](#aggregator-host)         | required | ""                                |
| [fluentd-sidecar-injector.h3poteto.dev/aggregator-port](#aggregator-port)         | optional | `24224`                           |
| [fluentd-sidecar-injector.h3poteto.dev/application-log-dir](#application-log-dir) | required | ""                                |
| [fluentd-sidecar-injector.h3poteto.dev/tag-prefix](#tag-prefix)                   | optional | ""                             |
| [fluentd-sidecar-injector.h3poteto.dev/custom-env](#custom-env)                   | optional | ""                                |
| [fluentd-sidecar-injector.h3poteto.dev/expose-port](#expose-port)                 | optional | ""                                |
| [fluentd-sidecar-injector.h3poteto.dev/config-volume](#config-volume)             | optional | ""                                |

These annotations are used when `collector` is `fluentd`.

| Name                                                                             | Required | Default                           |
| ---------------------------------------------------------------------------------| -------- | --------------------------------- |
| [fluentd-sidecar-injector.h3poteto.dev/send-timeout](#send-timeout)               | optional | `60s`                             |
| [fluentd-sidecar-injector.h3poteto.dev/recover-wait](#recover-wait)               | optional | `10s`                             |
| [fluentd-sidecar-injector.h3poteto.dev/hard-timeout](#hard-timeout)               | optional | `120s`                            |
| [fluentd-sidecar-injector.h3poteto.dev/time-key](#time-key)                       | optional | `time`                            |
| [fluentd-sidecar-injector.h3poteto.dev/time-format](#time-format)                 | optional | `%Y-%m-%dT%H:%M:%S%z`             |
| [fluentd-sidecar-injector.h3poteto.dev/log-format](#log-format)                   | optional | `json`                            |


These annotations are used when `collector` is `fluent-bit`.

| Name                                                                             | Required | Default                           |
| ---------------------------------------------------------------------------------| -------- | --------------------------------- |
| [fluentd-sidecar-injector.h3poteto.dev/refresh-interval](#refresh-interval)        | optional | `60`                              |
| [fluentd-sidecar-injector.h3poteto.dev/rotate-wait](#rotate-wait)                 | optional | `5`                               |

- <a name="injection">`fluentd-sidecar-injector.h3poteto.dev/injection`<a/> specifies whether enable or disable this injector. Please specify `enabled` if you want to enable.
- <a name="docker-image">`fluentd-sidecar-injector.h3poteto.dev/docker-image`</a> specifies sidecar docker image. Default is `ghcr.io/h3poteto/fluentd-forward:latest`.
- <a name="collector">`fluentd-sidecar-injector.h3poteto.dev/collector`</a> specifies collector name which is `fluentd` or `fluent-bit`. Default is `fluentd`. Specified collector is injected you pods.
- <a name="aggregator-host">`fluentd-sidecar-injector.h3poteto.dev/aggregator-host`</a> is used in [here](https://github.com/h3poteto/docker-fluentd-forward/blob/master/fluent.conf#L39). Default docker image forward received logs to another fluentd host. This parameter is required.
- <a name="aggregator-port">`fluentd-sidecar-injector.h3poteto.dev/aggregator-port`</a> is used in [here](https://github.com/h3poteto/docker-fluentd-forward/blob/master/fluent.conf#L40). Default is `24224`.
- <a name="application-log-dir">`fluentd-sidecar-injector.h3poteto.dev/application-log-dir`</a> specifies log directory where fluentd will watch. This directory is share between application container and sidecar fluentd container using volume mounts. This parameter is required.
- <a name="tag-prefix">`fluentd-sidecar-injector.h3poteto.dev/tag-prefix`</a> is prefix of received log's tag. It is used in [here](https://github.com/h3poteto/docker-fluentd-forward/blob/master/fluent.conf#L5).
- <a name="config-volume">`fluentd-sidecar-injector.h3poteto.dev/config-volume`</a> can read your own fluent.conf. If you specify `collector` to `fluent-bit`, `fluent-bit.conf` is read.
- <a name="custom-env">`fluentd-sidecar-injector.h3poteto.dev/custom-env`</a> is an option that allows users to set their own values ​​in fluent.conf. Use with config-volume option.
- <a name="expose-port">`fluentd-sidecar-injector.h3poteto.dev/expose-port`</a> is an option that users can set any port to expose fluentd container.
- <a name="send-timeout">`fluentd-sidecar-injector.h3poteto.dev/send-timeout`</a> is send timeout of fluentd configuration in [here](https://github.com/h3poteto/docker-fluentd-forward/blob/master/fluent.conf#L16). Default is `60s`.
- <a name="recover-wait">`fluentd-sidecar-injector.h3poteto.dev/recover-wait`</a> is used in [here](https://github.com/h3poteto/docker-fluentd-forward/blob/master/fluent.conf#L17). Default is `10s`.
- <a name="hard-timeout">`fluentd-sidecar-injector.h3poteto.dev/hard-timeout`</a> is timeout of fluentd configuration in [here](https://github.com/h3poteto/docker-fluentd-forward/blob/master/fluent.conf#L18). Default is `120s`.
- <a name="time-key">`fluentd-sidecar-injector.h3poteto.dev/time-key`</a> is fluentd configuration in [here](https://github.com/h3poteto/docker-fluentd-forward/blob/master/fluent.conf#L9). Default is `time`.
- <a name="time-format">`fluentd-sidecar-injector.h3poteto.dev/time-format`</a> is fluentd configuration in [here](https://github.com/h3poteto/docker-fluentd-forward/blob/master/fluent.conf#L10). Default is `%Y-%m-%dT%H:%M:%S%z`.
- <a name="log-format">`fluentd-sidecar-injector.h3poteto.dev/log-format`</a> is fluentd configuration in [here](https://github.com/h3poteto/docker-fluentd-forward/blob/master/fluent.conf#L7). Default is `json`.
- <a name="refresh-interval">`fluentd-sidecar-injector.h3poteto.dev/refresh-interval`</a> is fluent-bit configuration in [hrere](https://github.com/h3poteto/docker-fluentbit-forward/blob/master/fluent-bit.conf#L11). Default is `60` second.
- <a name="rotate-wait">`fluentd-sidecar-injector.h3poteto.dev/rotate-wait`</a> is fluent-bit configuration in [here](https://github.com/h3poteto/docker-fluentbit-forward/blob/master/fluent-bit.conf#L12). Default is `5` second.

### Fixed environment variables

The following values ​​will be set for each fluentd-sidecar.
You can use this value in your fluent.conf with config-volume option.

| Name                | Default                   |
| ------------------- | ------------------------- |
| NODE_NAME           | `spec.nodeName`           |
| POD_NAME            | `metadata.name`           |
| POD_NAMESPACE       | `metadata.namespace`      |
| POD_IP              | `status.podIP`            |
| POD_SERVICE_ACCOUNT | `spec.serviceAccountName` |
| CPU_RESOURCE        | `requests.cpu`            |
| CPU_LIMIT           | `limits.cpu`              |
| MEM_RESOURCE        | `requests.memory`         |
| MEM_LIMIT           | `limits.memory`           |

You can find out more about the values on [The Downward API](https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/#the-downward-api).

## Development
Please prepare a Kubernetes cluster to install this, and export `KUBECONFIG`.

```
$ export KUBECONFIG=$HOME/.kube/config
```

At first, please install CRDs.

```
$ make install
```

Next, please run controller in local.

```
$ make run
```

## License

The package is available as open source under the terms of the [MIT License](https://opensource.org/licenses/MIT).
