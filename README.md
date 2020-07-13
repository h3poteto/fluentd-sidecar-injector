[![CircleCI](https://circleci.com/gh/h3poteto/fluentd-sidecar-injector.svg?style=svg)](https://circleci.com/gh/h3poteto/fluentd-sidecar-injector)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/h3poteto/fluentd-sidecar-injector?sort=semver&style=flat-square)
[![Dependabot](https://img.shields.io/badge/Dependabot-enabled-blue.svg)](https://dependabot.com)

# fluentd-sidecar-injector

`fluentd-sidecar-injector` is a webhook server for kubernetes admission webhook. This server inject fluentd container as sidecar for specified Pod using mutation webhook. The feature is

- Automatically sidecar injection
- You can control injection using Pod's annotations
- You can change fluentd docker image to be injected

## Usage

After you install this webhook server, fluentd sidecar containers are automatically injected. If you provide a deployment:

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
      labels:
        app: nginx-test
    spec:
      containers:
        - name: nginx
          image: nginx:latest
```

fluentd is injected for this Pod.

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
    Image:          h3poteto/fluentd-forward:latest
    Image ID:       docker-pullable://h3poteto/fluentd-forward@sha256:5d93af333ad9fefbfcb8013d20834fd89c2bbd3fe8b9b9bfa620ded29d7b3205
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
      format "ltsv"
      tag "app.*"
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

## Install

```sh
$ git clone https://github.com/h3poteto/fluentd-sidecar-injector.git
$ cd fluentd-sidecar-injector
```

At first, please use `make` to generate kustomize template files.

```sh
$ make build NAMESPACE=kube-system
```

You can specify `NAMESPACE` where you want to install this webhook server. It works fine with any namespace. Please customize generated kustomization files if you want.

Next, please install it.

```sh
$ kubectl apply -k ./install/kustomize
```

## Annotations

Please specify these annotations to your pods like [this](example/deployment.yaml).

| Name                                                                              | Required | Default                           |
| --------------------------------------------------------------------------------- | -------- | --------------------------------- |
| [fluentd-sidecar-injector.h3poteto.dev/injection](#injection)                     | optional | ""                                |
| [fluentd-sidecar-injector.h3poteto.dev/docker-image](#docker-image)               | optional | `h3poteto/fluentd-forward:latest` |
| [fluentd-sidecar-injector.h3poteto.dev/aggregator-host](#aggregator-host)         | required | ""                                |
| [fluentd-sidecar-injector.h3poteto.dev/aggregator-port](#aggregator-port)         | optional | `24224`                           |
| [fluentd-sidecar-injector.h3poteto.dev/application-log-dir](#application-log-dir) | required | ""                                |
| [fluentd-sidecar-injector.h3poteto.dev/send-timeout](#send-timeout)               | optional | `60s`                             |
| [fluentd-sidecar-injector.h3poteto.dev/recover-wait](#recover-wait)               | optional | `10s`                             |
| [fluentd-sidecar-injector.h3poteto.dev/hard-timeout](#hard-timeout)               | optional | `120s`                            |
| [fluentd-sidecar-injector.h3poteto.dev/tag-prefix](#tag-prefix)                   | optional | `app`                             |
| [fluentd-sidecar-injector.h3poteto.dev/time-key](#time-key)                       | optional | `time`                            |
| [fluentd-sidecar-injector.h3poteto.dev/time-format](#time-format)                 | optional | `%Y-%m-%dT%H:%M:%S%z`             |
| [fluentd-sidecar-injector.h3poteto.dev/log-format](#log-format)                   | optional | `json`                            |
| [fluentd-sidecar-injector.h3poteto.dev/config-volume](#config-volume)             | optional | ""                                |

- <a name="injection">`fluentd-sidecar-injector.h3poteto.dev/injection`<a/> specifies whether enable or disable this injector. Please specify `enabled` if you want to enable.

- <a name="docker-image">`fluentd-sidecar-injector.h3poteto.dev/docker-image`</a> specifies sidecar docker image. Default is `h3poteto/fluentd-forward:latest`.
- <a name="aggregator-host">`fluentd-sidecar-injector.h3poteto.dev/aggregator-host`</a> is used in [here](https://github.com/h3poteto/docker-fluentd-forward/blob/master/fluent.conf#L37). Default docker image forward received logs to another fluentd host. This parameter is required.
- <a name="aggregator-port">`fluentd-sidecar-injector.h3poteto.dev/aggregator-port`</a> is used in [here](https://github.com/h3poteto/docker-fluentd-forward/blob/master/fluent.conf#L38). Default is `24224`.
- <a name="application-log-dir">`fluentd-sidecar-injector.h3poteto.dev/application-log-dir`</a> specifies log directory where fluentd will watch. This directory is share between application container and sidecar fluentd container using volume mounts. This parameter is required.
- <a name="send-timeout">`fluentd-sidecar-injector.h3poteto.dev/send-timeout`</a> is send timeout of fluentd configuration in [here](https://github.com/h3poteto/docker-fluentd-forward/blob/master/fluent.conf#L14). Default is `60s`.
- <a name="recover-wait">`fluentd-sidecar-injector.h3poteto.dev/recover-wait`</a> is used in [here](https://github.com/h3poteto/docker-fluentd-forward/blob/master/fluent.conf#L15). Default is `10s`.
- <a name="hard-timeout">`fluentd-sidecar-injector.h3poteto.dev/hard-timeout`</a> is timeout of fluentd configuration in [here](https://github.com/h3poteto/docker-fluentd-forward/blob/master/fluent.conf#L16). Default is `120s`.
- <a name="tag-prefix">`fluentd-sidecar-injector.h3poteto.dev/tag-prefix`</a> is prefix of received log's tag. Default is `app`. It is used in [here](https://github.com/h3poteto/docker-fluentd-forward/blob/master/fluent.conf#L9).
- <a name="time-key">`fluentd-sidecar-injector.h3poteto.dev/time-key`</a> is fluentd configuration in [here](https://github.com/h3poteto/docker-fluentd-forward/blob/master/fluent.conf#L6). Default is `time`.
- <a name="time-format">`fluentd-sidecar-injector.h3poteto.dev/time-format`</a> is fluentd configuration in [here](https://github.com/h3poteto/docker-fluentd-forward/blob/master/fluent.conf#L7). Default is `%Y-%m-%dT%H:%M:%S%z`.
- <a name="log-format">`fluentd-sidecar-injector.h3poteto.dev/log-format`</a> is fluentd configuration in [here](https://github.com/h3poteto/docker-fluentd-forward/blob/master/fluent.conf#L5). Default is `json`.
- <a name="config-volume">`fluentd-sidecar-injector.h3poteto.dev/config-volume`</a> can read your own fluent.conf.

## Environment variables

If you use same parameters for all sidecar fluentd containers which are injected by this webhook, you can set the parameters with environment variables. If you want to specify these environment variables, please customize [kustomize template](install/kustomize/base/deployment.yaml).

| Name                                                | Default                           |
| --------------------------------------------------- | --------------------------------- |
| [FLUENTD_DOCKER_IMAGE](#docker-image)               | `h3poteto/fluentd-forward:latest` |
| [FLUENTD_AGGREGATOR_HOST](#aggregator-host)         | ""                                |
| [FLUENTD_AGGREGATOR_PORT](#aggregator-port)         | `24224`                           |
| [FLUENTD_APPLICATION_LOG_DIR](#application-log-dir) | ""                                |
| [FLUENTD_TAG_PREFIX](#tag-prefix)                   | `app`                             |
| [FLUENTD_TIME_KEY](#time-key)                       | `time`                            |
| [FLUENTD_TIME_FORMAT](#time-format)                 | `%Y-%m-%dT%H:%M:%S%z`             |

Note: these parameters will be overrided with Pod annotations if you set.

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

## License

The package is available as open source under the terms of the [MIT License](https://opensource.org/licenses/MIT).
