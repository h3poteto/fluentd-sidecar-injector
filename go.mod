module github.com/h3poteto/fluentd-sidecar-injector

go 1.13

require (
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/sirupsen/logrus v1.5.0
	github.com/slok/kubewebhook v0.8.0
	github.com/spf13/cobra v0.0.6
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.4-beta.0
)
