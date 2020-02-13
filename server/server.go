package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/kelseyhightower/envconfig"
	webhookhttp "github.com/slok/kubewebhook/pkg/http"
	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/webhook/mutating"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Env struct {
	DockerImage       string `envconfig:"DOCKER_IMAGE" default:"h3poteto/fluentd-forward:latest"`
	ApplicationLogDir string `envconfig:"APPLICATION_LOG_DIR"`
	TimeKey           string `envconfig:"TIME_KEY"`
	TagPrefix         string `envconfig:"TAG_PREFIX"`
	AggregatorHost    string `envconfig:"AGGREGATOR_HOST"`
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

	if pod.Annotations["fluentd-sidecar-injector.h3poteto.dev/injection"] != "enabled" {
		return false, nil
	}

	var fluentdEnv Env
	envconfig.Process("fluentd", &fluentdEnv)

	dockerImage := fluentdEnv.DockerImage
	if value, ok := pod.Annotations["fluentd-sidecar-injector.h3poteto.dev/docker-image"]; ok {
		dockerImage = value
	}

	volumeName := "fluentd-sidecar-injector-logs"
	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	sidecar := corev1.Container{
		Name:  "fluentd-sidecar",
		Image: dockerImage,
		Resources: corev1.ResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceMemory: *resource.NewQuantity(200*1024*1024, resource.BinarySI),
				corev1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
			},
			Limits: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceMemory: *resource.NewQuantity(1000*1024*1024, resource.BinarySI),
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			corev1.VolumeMount{
				Name:      volumeName,
				ReadOnly:  false,
				MountPath: "/app",
			},
		},
	}

	// Override env with fluentdEnv and Pod's annotations.
	aggregatorHost := fluentdEnv.AggregatorHost
	if value, ok := pod.Annotations["fluentd-sidecar-injector.h3poteto.dev/aggregator-host"]; ok {
		aggregatorHost = value
	}

	if len(aggregatorHost) > 0 {
		sidecar.Env = append(sidecar.Env, corev1.EnvVar{
			Name:  "AGGREGATOR_HOST",
			Value: aggregatorHost,
		})
	}

	applicationLogDir := fluentdEnv.ApplicationLogDir
	if value, ok := pod.Annotations["fluentd-sidecar-injector.h3poteto.dev/application-log-dir"]; ok {
		applicationLogDir = value
	}
	if len(applicationLogDir) == 0 {
		return false, errors.New("application log dir is required")
	}
	sidecar.Env = append(sidecar.Env, corev1.EnvVar{
		Name:  "APPLICATION_LOG_DIR",
		Value: applicationLogDir,
	})

	volumeMount := corev1.VolumeMount{
		Name:      volumeName,
		ReadOnly:  false,
		MountPath: applicationLogDir,
	}

	tagPrefix := fluentdEnv.TagPrefix
	if value, ok := pod.Annotations["fluentd-sidecar-injector.h3poteto.dev/tag-prefix"]; ok {
		tagPrefix = value
	}
	if len(tagPrefix) > 0 {
		sidecar.Env = append(sidecar.Env, corev1.EnvVar{
			Name:  "TAG_PREFIX",
			Value: tagPrefix,
		})
	}

	timeKey := fluentdEnv.TimeKey
	if value, ok := pod.Annotations["fluentd-sidecar-injector.h3poteto.dev/time-key"]; ok {
		timeKey = value
	}
	if len(timeKey) > 0 {
		sidecar.Env = append(sidecar.Env, corev1.EnvVar{
			Name:  "TIME_KEY",
			Value: timeKey,
		})
	}

	pod.Spec.Containers = append(pod.Spec.Containers, sidecar)

	// Inject volume mount for all containers in the pod.
	for _, container := range pod.Spec.Containers {
		container.VolumeMounts = append(container.VolumeMounts, volumeMount)
	}

	return false, nil
}
