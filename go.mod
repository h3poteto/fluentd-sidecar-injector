module github.com/h3poteto/fluentd-sidecar-injector

go 1.15

require (
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/sirupsen/logrus v1.7.0
	github.com/slok/kubewebhook v0.11.0
	github.com/spf13/cobra v1.1.1
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.3
	k8s.io/klog v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v3 v3.0.0 // indirect
)
