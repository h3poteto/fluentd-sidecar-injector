namespace: NAMESPACE

bases:
  - ./base

patches:
  - mutating-webhook-configuration.yaml

imageTags:
  - name: h3poteto/fluentd-sidecar-injector
    newTag: v0.1.0