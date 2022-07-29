module github.com/h3poteto/fluentd-sidecar-injector

go 1.16

require (
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/onsi/ginkgo/v2 v2.1.4
	github.com/onsi/gomega v1.20.0
	github.com/sirupsen/logrus v1.8.1
	github.com/slok/kubewebhook v0.11.0
	github.com/spf13/cobra v1.3.0
	github.com/spf13/viper v1.10.1
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v0.22.6
	k8s.io/apimachinery v0.22.6
	k8s.io/client-go v0.22.6
	k8s.io/klog/v2 v2.40.1
	k8s.io/utils v0.0.0-20220127004650-9b3446523e65
)
