namespace: NAMESPACE

bases:
  - ./base

patches:
  - mutating-webhook-configuration.yaml

imageTags:
  - name: ghcr.io/h3poteto/fluentd-sidecar-injector
    newTag: latest
