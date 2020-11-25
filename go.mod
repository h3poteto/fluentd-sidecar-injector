module github.com/h3poteto/fluentd-sidecar-injector

go 1.15

require (
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/sirupsen/logrus v1.7.0
	github.com/slok/kubewebhook v0.11.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.4
	k8s.io/client-go v0.19.3
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.4.0
	k8s.io/utils v0.0.0-20200729134348-d5654de09c73
	sigs.k8s.io/structured-merge-diff/v3 v3.0.0 // indirect
)
