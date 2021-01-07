package sidecarinjector

import (
	"crypto/tls"
	"testing"

	sidecarinjectorv1alpha1 "github.com/h3poteto/fluentd-sidecar-injector/pkg/apis/sidecarinjectorcontroller/v1alpha1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFluentDNewDeployment(t *testing.T) {
	manifest := &sidecarinjectorv1alpha1.SidecarInjector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unit-test",
			Namespace: "default",
		},
		Spec: sidecarinjectorv1alpha1.SidecarInjectorSpec{
			Collector: "fluentd",
			FluentD: &sidecarinjectorv1alpha1.FluentDSpec{
				DockerImage:       "my-fluentd-image:some-tag",
				AggregatorHost:    "my-aggregator-host.local",
				AggregatorPort:    24224,
				ApplicationLogDir: "/var/log/my-logs",
				TagPrefix:         "my-tag",
				TimeKey:           "time",
				TimeFormat:        "%Y-%m-%dT%H:%M:%S",
			},
			FluentBit: nil,
		},
	}

	deployment := newDeployment(manifest, "test-secret", "my-injector-image:tag")

	if deployment.Name != "unit-test-handler" {
		t.Errorf("Deployment name is not matched: %s", deployment.Name)
	}
	if deployment.Namespace != "default" {
		t.Errorf("Deployment namespace is not matched: %s", deployment.Namespace)
	}
	if *deployment.Spec.Replicas != int32(2) {
		t.Errorf("Deployment replicas is not matched: %d", *deployment.Spec.Replicas)
	}
	if deployment.Spec.Template.Spec.Volumes[0].Name != "webhook-certs" {
		t.Errorf("Deployment volumes are not matched: %s", deployment.Spec.Template.Spec.Volumes[0].Name)
	}
	if deployment.Spec.Template.Spec.Volumes[0].VolumeSource.Secret.SecretName != "test-secret" {
		t.Errorf("Deployment volume secret is not matched: %s", deployment.Spec.Template.Spec.Volumes[0].VolumeSource.Secret.SecretName)
	}
	if deployment.Spec.Template.Spec.Containers[0].Name != "webhook-handler" {
		t.Errorf("Deployment container name is not matched: %s", deployment.Spec.Template.Spec.Containers[0].Name)
	}
	if deployment.Spec.Template.Spec.Containers[0].Image != "my-injector-image:tag" {
		t.Errorf("Deployment container image is not matched: %s", deployment.Spec.Template.Spec.Containers[0].Image)
	}

	if dockerImage := findEnv(deployment.Spec.Template.Spec.Containers[0].Env, "FLUENTD_DOCKER_IMAGE"); dockerImage == nil || dockerImage.Value != "my-fluentd-image:some-tag" {
		t.Errorf("Container env docker image is not matched: %v", dockerImage)
	}
	if aggregatorHost := findEnv(deployment.Spec.Template.Spec.Containers[0].Env, "FLUENTD_AGGREGATOR_HOST"); aggregatorHost == nil || aggregatorHost.Value != "my-aggregator-host.local" {
		t.Errorf("Container env aggregator host is not matched: %v", aggregatorHost)
	}
	if aggregatorPort := findEnv(deployment.Spec.Template.Spec.Containers[0].Env, "FLUENTD_AGGREGATOR_PORT"); aggregatorPort == nil || aggregatorPort.Value != "24224" {
		t.Errorf("Container env aggregator port is not matched: %v", aggregatorPort)
	}
	if aggregatorLogDir := findEnv(deployment.Spec.Template.Spec.Containers[0].Env, "FLUENTD_APPLICATION_LOG_DIR"); aggregatorLogDir == nil || aggregatorLogDir.Value != "/var/log/my-logs" {
		t.Errorf("Container env aggregator log dir is not matched: %v", aggregatorLogDir)
	}
	if tagPrefix := findEnv(deployment.Spec.Template.Spec.Containers[0].Env, "FLUENTD_TAG_PREFIX"); tagPrefix == nil || tagPrefix.Value != "my-tag" {
		t.Errorf("Container env tag prefix is not matched: %v", tagPrefix)
	}
	if timeKey := findEnv(deployment.Spec.Template.Spec.Containers[0].Env, "FLUENTD_TIME_KEY"); timeKey == nil || timeKey.Value != "time" {
		t.Errorf("Container env time key is not matched: %v", timeKey)
	}
	if timeFormat := findEnv(deployment.Spec.Template.Spec.Containers[0].Env, "FLUENTD_TIME_FORMAT"); timeFormat == nil || timeFormat.Value != "%Y-%m-%dT%H:%M:%S" {
		t.Errorf("Container env time format is not matched: %v", timeFormat)
	}
}

func TestFluentBitNewDeployment(t *testing.T) {
	manifest := &sidecarinjectorv1alpha1.SidecarInjector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unit-test",
			Namespace: "default",
		},
		Spec: sidecarinjectorv1alpha1.SidecarInjectorSpec{
			Collector: "fluent-bit",
			FluentD:   nil,
			FluentBit: &sidecarinjectorv1alpha1.FluentBitSpec{
				DockerImage:       "my-fluent-bit-image:some-tag",
				AggregatorHost:    "my-aggregator-host.local",
				AggregatorPort:    24224,
				ApplicationLogDir: "/var/log/my-logs",
				TagPrefix:         "my-tag",
			},
		},
	}

	deployment := newDeployment(manifest, "test-secret", "my-injector-image:tag")

	if deployment.Name != "unit-test-handler" {
		t.Errorf("Deployment name is not matched: %s", deployment.Name)
	}
	if deployment.Namespace != "default" {
		t.Errorf("Deployment namespace is not matched: %s", deployment.Namespace)
	}
	if *deployment.Spec.Replicas != int32(2) {
		t.Errorf("Deployment replicas is not matched: %d", *deployment.Spec.Replicas)
	}
	if deployment.Spec.Template.Spec.Volumes[0].Name != "webhook-certs" {
		t.Errorf("Deployment volumes are not matched: %s", deployment.Spec.Template.Spec.Volumes[0].Name)
	}
	if deployment.Spec.Template.Spec.Volumes[0].VolumeSource.Secret.SecretName != "test-secret" {
		t.Errorf("Deployment volume secret is not matched: %s", deployment.Spec.Template.Spec.Volumes[0].VolumeSource.Secret.SecretName)
	}
	if deployment.Spec.Template.Spec.Containers[0].Name != "webhook-handler" {
		t.Errorf("Deployment container name is not matched: %s", deployment.Spec.Template.Spec.Containers[0].Name)
	}
	if deployment.Spec.Template.Spec.Containers[0].Image != "my-injector-image:tag" {
		t.Errorf("Deployment container image is not matched: %s", deployment.Spec.Template.Spec.Containers[0].Image)
	}
	if dockerImage := findEnv(deployment.Spec.Template.Spec.Containers[0].Env, "FLUENTBIT_DOCKER_IMAGE"); dockerImage == nil || dockerImage.Value != "my-fluent-bit-image:some-tag" {
		t.Errorf("Container env docker image is not matched: %v", dockerImage)
	}
	if aggregatorHost := findEnv(deployment.Spec.Template.Spec.Containers[0].Env, "FLUENTBIT_AGGREGATOR_HOST"); aggregatorHost == nil || aggregatorHost.Value != "my-aggregator-host.local" {
		t.Errorf("Container env aggregator host is not matched: %v", aggregatorHost)
	}
	if aggregatorPort := findEnv(deployment.Spec.Template.Spec.Containers[0].Env, "FLUENTBIT_AGGREGATOR_PORT"); aggregatorPort == nil || aggregatorPort.Value != "24224" {
		t.Errorf("Container env aggregator port is not matched: %v", aggregatorPort)
	}
	if aggregatorLogDir := findEnv(deployment.Spec.Template.Spec.Containers[0].Env, "FLUENTBIT_APPLICATION_LOG_DIR"); aggregatorLogDir == nil || aggregatorLogDir.Value != "/var/log/my-logs" {
		t.Errorf("Container env aggregator log dir is not matched: %v", aggregatorLogDir)
	}
	if tagPrefix := findEnv(deployment.Spec.Template.Spec.Containers[0].Env, "FLUENTBIT_TAG_PREFIX"); tagPrefix == nil || tagPrefix.Value != "my-tag" {
		t.Errorf("Container env tag prefix is not matched: %v", tagPrefix)
	}
}

func findEnv(env []corev1.EnvVar, targetName string) *corev1.EnvVar {
	for i := range env {
		if env[i].Name == targetName {
			return &env[i]
		}
	}
	return nil
}

func TestNewCertificates(t *testing.T) {
	serviceName := "my-cluster"
	namespace := "kube-system"
	key, cert, err := NewCertificates(serviceName, namespace)
	if err != nil {
		t.Error(err)
	}
	certificate, err := tls.X509KeyPair(cert, key)
	if err != nil {
		t.Error(err)
	}
	if certificate.Leaf != nil {
		t.Errorf("Failed to parse certificate: %v", certificate)
	}
}

func TestNewMutatingWebhookConfiguration(t *testing.T) {
	serviceName := "my-cluster"
	namespace := "kube-system"
	_, cert, err := NewCertificates(serviceName, namespace)
	if err != nil {
		t.Error(err)
	}

	injector := &sidecarinjectorv1alpha1.SidecarInjector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: namespace,
		},
		Spec: sidecarinjectorv1alpha1.SidecarInjectorSpec{
			Collector: "fluent-bit",
			FluentD:   nil,
			FluentBit: &sidecarinjectorv1alpha1.FluentBitSpec{},
		},
	}

	conf := newMutatingWebhookConfiguration(injector, serviceName, serviceName, cert)
	if conf.Namespace != namespace {
		t.Errorf("Namespace is not matched: %s", conf.Namespace)
	}
	if conf.Webhooks[0].Name != serviceName+"."+namespace+".svc" {
		t.Errorf("Webhook name is not matched: %s", conf.Webhooks[0].Name)
	}
	if string(conf.Webhooks[0].ClientConfig.CABundle) != string(cert) {
		t.Errorf("Webhook CABundle is not matched: %v", conf.Webhooks[0].ClientConfig.CABundle)
	}
	if *conf.Webhooks[0].ClientConfig.Service.Path != "/mutate" {
		t.Errorf("Webhook Service path is not matched: %s", *conf.Webhooks[0].ClientConfig.Service.Path)
	}
	if conf.Webhooks[0].ClientConfig.Service.Name != serviceName {
		t.Errorf("Webhook Service name is not matched: %s", conf.Webhooks[0].ClientConfig.Service.Name)
	}
	if conf.Webhooks[0].ClientConfig.Service.Namespace != namespace {
		t.Errorf("Webhook Service namespace is not matched: %s", conf.Webhooks[0].ClientConfig.Service.Namespace)
	}
	if *conf.Webhooks[0].FailurePolicy != admissionregistrationv1.Ignore {
		t.Errorf("Webhook FailurePolicy is not matched: %v", *conf.Webhooks[0].FailurePolicy)
	}
	if conf.Webhooks[0].AdmissionReviewVersions[0] != "v1beta1" {
		t.Errorf("Webhook AdmissionReviewVersions is not matched: %v", conf.Webhooks[0].AdmissionReviewVersions)
	}
}
