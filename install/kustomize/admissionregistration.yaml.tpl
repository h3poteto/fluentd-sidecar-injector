apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: fluentd-sidecar-injector
webhooks:
  - name: fluentd-sidecar-injector.h3poteto.dev
    clientConfig:
      service:
        namespace: NAMESPACE
      caBundle: CA_BUNDLE