package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kelseyhightower/envconfig"
	webhookhttp "github.com/slok/kubewebhook/pkg/http"
	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/webhook/mutating"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Env struct {
	DockerImage        string `envconfig:"DOCKER_IMAGE" default:"h3poteto/fluentd-forward:latest"`
	ApplicationLogPath string `envconfig:"APPLICATION_LOG_PATH" default:"/app"`
	TimeKey            string `envconfig:"TIME_KEY" default:"time"`
	TagPrefix          string `envconfig:"TAG_PREFIX"`
	AggregatorHost     string `envconfig:"AGGREGATOR_HOST"`
}

func StartServer(tlsCertFile, tlsKeyFile string) error {
	logger := &log.Std{Debug: true}

	mutator := mutating.MutatorFunc(sidecarInjectMutator)

	config := mutating.WebhookConfig{
		Name: "fluentdSidecarInjector",
		Obj:  &corev1.Pod{},
	}
	webhook, err := mutating.NewWebhook(config, mutator, nil, nil, logger)
	if err != nil {
		return fmt.Errorf("Failed to create webhook: %s", err)
	}

	handler, err := webhookhttp.HandlerFor(webhook)
	if err != nil {
		return fmt.Errorf("Failed to create webhook handler: %s", err)
	}

	logger.Infof("Listing on :8080")
	err = http.ListenAndServeTLS(":8080", tlsCertFile, tlsKeyFile, handler)
	if err != nil {
		return fmt.Errorf("Failed to start server: %s", err)
	}

	return nil

}

func sidecarInjectMutator(_ context.Context, obj metav1.Object) (stop bool, err error) {
	pod, ok := obj.(*corev1.Pod)

	if !ok {
		return false, nil
	}

	if pod.Annotations["h3poteto.dev.fluentd-sidecar-injection"] != "enabled" {
		return false, nil
	}

	var fluentdEnv Env
	envconfig.Process("fluentd", &fluentdEnv)

	sidecar := corev1.Container{
		Name:  "fluentd-sidecar",
		Image: fluentdEnv.DockerImage,
	}

	// Override env with fluentdEnv.
	if len(fluentdEnv.AggregatorHost) > 0 {
		sidecar.Env = append(sidecar.Env, corev1.EnvVar{
			Name:  "AGGREGATOR_HOST",
			Value: fluentdEnv.AggregatorHost,
		})
	}
	if len(fluentdEnv.ApplicationLogPath) > 0 {
		sidecar.Env = append(sidecar.Env, corev1.EnvVar{
			Name:  "APPLICATION_LOG_PATH",
			Value: fluentdEnv.ApplicationLogPath,
		})
	}
	if len(fluentdEnv.TagPrefix) > 0 {
		sidecar.Env = append(sidecar.Env, corev1.EnvVar{
			Name:  "TAG_PREFIX",
			Value: fluentdEnv.TagPrefix,
		})
	}
	if len(fluentdEnv.TimeKey) > 0 {
		sidecar.Env = append(sidecar.Env, corev1.EnvVar{
			Name:  "TIME_KEY",
			Value: fluentdEnv.TimeKey,
		})
	}

	// TODO: Override env with Pod's annotations.

	pod.Spec.Containers = append(pod.Spec.Containers, sidecar)

	return false, nil
}
