module github.com/h3poteto/fluentd-sidecar-injector

go 1.15

require (
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/nxadm/tail v1.4.6 // indirect
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.10.5
	github.com/sirupsen/logrus v1.7.0
	github.com/slok/kubewebhook v0.11.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/klog/v2 v2.8.0
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
)
