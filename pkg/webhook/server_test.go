package webhook

import (
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInjectFluentD(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "",
			Namespace: "",
			Annotations: map[string]string{
				annotationPrefix + "/injection":           "enabled",
				annotationPrefix + "/aggregator-host":     "my-aggregator.local",
				annotationPrefix + "/application-log-dir": "/var/log/nginx",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:latest",
				},
			},
		},
	}

	result, err := injectFluentD(pod)
	if err != nil {
		t.Error(err)
	}
	if result {
		t.Error("Could not inject sidecar")
	}
	if pod.Spec.Volumes[0].Name != VolumeName || pod.Spec.Volumes[0].VolumeSource.EmptyDir == nil {
		t.Errorf("Failed to append volumes to pod: %#v", pod.Spec.Volumes)
	}
	if len(pod.Spec.Containers) != 2 {
		t.Errorf("Failed to append sidecar container: %#v", pod.Spec.Containers)
	}
	container := findContainer(pod.Spec.Containers, ContainerName)
	if container == nil {
		t.Errorf("Failed to injecto sidecar container: %#v", pod.Spec.Containers)
		return
	}
	if container.Image != "ghcr.io/h3poteto/fluentd-forward:latest" {
		t.Errorf("Container image is not matched: %s", container.Image)
	}

	if sendTimeout := findEnv(container.Env, "SEND_TIMEOUT"); sendTimeout.Value != "60s" {
		t.Errorf("Container env send timeout is not matched: %v", sendTimeout)
	}
	if recoverWait := findEnv(container.Env, "RECOVER_WAIT"); recoverWait.Value != "10s" {
		t.Errorf("Container env recover wait is not matched: %v", recoverWait)
	}
	if hardTimeout := findEnv(container.Env, "HARD_TIMEOUT"); hardTimeout.Value != "120s" {
		t.Errorf("Container env hard timeout is not matched: %v", hardTimeout)
	}

	if timeFormat := findEnv(container.Env, "TIME_FORMAT"); timeFormat.Value != "%Y-%m-%dT%H:%M:%S%z" {
		t.Errorf("Container env time format is not matched: %v", timeFormat)
	}
	if timeKey := findEnv(container.Env, "TIME_KEY"); timeKey.Value != "time" {
		t.Errorf("Container env time key is not matched: %v", timeKey)
	}
	if tagPrefix := findEnv(container.Env, "TAG_PREFIX"); tagPrefix.Value != "app" {
		t.Errorf("Container env tag prefix is not matched: %v", tagPrefix)
	}
	if aggregatorPort := findEnv(container.Env, "AGGREGATOR_PORT"); aggregatorPort.Value != "24224" {
		t.Errorf("Container env aggregator port is not matched: %v", aggregatorPort)
	}
	if logFormat := findEnv(container.Env, "LOG_FORMAT"); logFormat.Value != "json" {
		t.Errorf("Container env log format is not matched: %v", logFormat)
	}
	if logDir := findEnv(container.Env, "APPLICATION_LOG_DIR"); logDir.Value != "/var/log/nginx" {
		t.Errorf("Container env log dir is not matched: %v", logDir)
	}

	if container.VolumeMounts[0].MountPath != "/var/log/nginx" {
		t.Errorf("Container volume mount path is not matched: %v", container.VolumeMounts[0])
	}
}

func TestInjectFluentDDWithAnnotations(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "",
			Namespace: "",
			Annotations: map[string]string{
				annotationPrefix + "/injection":           "enabled",
				annotationPrefix + "/aggregator-host":     "my-aggregator.local",
				annotationPrefix + "/aggregator-port":     "24223",
				annotationPrefix + "/application-log-dir": "/var/log/nginx",
				annotationPrefix + "/docker-image":        "my-fluentd:latest",
				annotationPrefix + "/expose-port":         "80",
				annotationPrefix + "/send-timeout":        "30s",
				annotationPrefix + "/recover-wait":        "15s",
				annotationPrefix + "/hard-timeout":        "60s",
				annotationPrefix + "/log-format":          "nginx",
				annotationPrefix + "/config-volume":       "my-custom-volume",
				annotationPrefix + "/tag-prefix":          "my-app",
				annotationPrefix + "/time-key":            "timestamp",
				annotationPrefix + "/time-format":         "%Y/%m/%d %H:%M:%S",
			},
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "my-custom-volume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "some-config",
							},
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:latest",
				},
			},
		},
	}

	result, err := injectFluentD(pod)
	if err != nil {
		t.Error(err)
	}
	if result {
		t.Error("Could not inject sidecar")
	}

	if v := findVolume(pod.Spec.Volumes, VolumeName); v == nil || v.VolumeSource.EmptyDir == nil {
		t.Errorf("Failed to append volumes to pod: %#v", pod.Spec.Volumes)
	}
	if len(pod.Spec.Containers) != 2 {
		t.Errorf("Failed to append sidecar container: %#v", pod.Spec.Containers)
	}
	container := findContainer(pod.Spec.Containers, ContainerName)
	if container == nil {
		t.Errorf("Failed to injecto sidecar container: %#v", pod.Spec.Containers)
		return
	}
	if container.Image != "my-fluentd:latest" {
		t.Errorf("Container image is not matched: %s", container.Image)
	}
	if container.Ports[0].ContainerPort != int32(80) {
		t.Errorf("Container port is not matched: %d", container.Ports[0].ContainerPort)
	}

	if sendTimeout := findEnv(container.Env, "SEND_TIMEOUT"); sendTimeout.Value != "30s" {
		t.Errorf("Container env send timeout is not matched: %v", sendTimeout)
	}
	if recoverWait := findEnv(container.Env, "RECOVER_WAIT"); recoverWait.Value != "15s" {
		t.Errorf("Container env recover wait is not matched: %v", recoverWait)
	}
	if hardTimeout := findEnv(container.Env, "HARD_TIMEOUT"); hardTimeout.Value != "60s" {
		t.Errorf("Container env hard timeout is not matched: %v", hardTimeout)
	}

	if timeFormat := findEnv(container.Env, "TIME_FORMAT"); timeFormat.Value != "%Y/%m/%d %H:%M:%S" {
		t.Errorf("Container env time format is not matched: %v", timeFormat)
	}
	if timeKey := findEnv(container.Env, "TIME_KEY"); timeKey.Value != "timestamp" {
		t.Errorf("Container env time key is not matched: %v", timeKey)
	}
	if tagPrefix := findEnv(container.Env, "TAG_PREFIX"); tagPrefix.Value != "my-app" {
		t.Errorf("Container env tag prefix is not matched: %v", tagPrefix)
	}
	if aggregatorHost := findEnv(container.Env, "AGGREGATOR_HOST"); aggregatorHost.Value != "my-aggregator.local" {
		t.Errorf("Container env aggregator host is not matched: %v", aggregatorHost)
	}
	if aggregatorPort := findEnv(container.Env, "AGGREGATOR_PORT"); aggregatorPort.Value != "24223" {
		t.Errorf("Container env aggregator port is not matched: %v", aggregatorPort)
	}
	if logFormat := findEnv(container.Env, "LOG_FORMAT"); logFormat.Value != "nginx" {
		t.Errorf("Container env log format is not matched: %v", logFormat)
	}
	if logDir := findEnv(container.Env, "APPLICATION_LOG_DIR"); logDir.Value != "/var/log/nginx" {
		t.Errorf("Container env log dir is not matched: %v", logDir)
	}

	if container.VolumeMounts[0].MountPath != "/var/log/nginx" {
		t.Errorf("Container volume mount path is not matched: %v", container.VolumeMounts[0])
	}
}

func TestInjectFluentDWithEnv(t *testing.T) {
	os.Setenv("DOCKER_IMAGE", "my-fluentd-image:latest")
	os.Setenv("APPLICATION_LOG_DIR", "/var/log/nginx")
	os.Setenv("AGGREGATOR_HOST", "my-aggregator.local")
	os.Setenv("AGGREGATOR_PORT", "24223")
	os.Setenv("TIME_FORMAT", "%Y/%m/%d %H:%M:%S")
	os.Setenv("TIME_KEY", "timestamp")
	os.Setenv("TAG_PREFIX", "my-app")
	os.Setenv("LOG_FORMAT", "nginx")

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "",
			Namespace: "",
			Annotations: map[string]string{
				annotationPrefix + "/injection": "enabled",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:latest",
				},
			},
		},
	}

	result, err := injectFluentD(pod)
	if err != nil {
		t.Error(err)
	}
	if result {
		t.Error("Could not inject sidecar")
	}

	if v := findVolume(pod.Spec.Volumes, VolumeName); v == nil || v.VolumeSource.EmptyDir == nil {
		t.Errorf("Failed to append volumes to pod: %#v", pod.Spec.Volumes)
	}
	if len(pod.Spec.Containers) != 2 {
		t.Errorf("Failed to append sidecar container: %#v", pod.Spec.Containers)
	}
	container := findContainer(pod.Spec.Containers, ContainerName)
	if container == nil {
		t.Errorf("Failed to injecto sidecar container: %#v", pod.Spec.Containers)
		return
	}
	if container.Image != "my-fluentd-image:latest" {
		t.Errorf("Container image is not matched: %s", container.Image)
	}

	if timeFormat := findEnv(container.Env, "TIME_FORMAT"); timeFormat.Value != "%Y/%m/%d %H:%M:%S" {
		t.Errorf("Container env time format is not matched: %v", timeFormat)
	}
	if timeKey := findEnv(container.Env, "TIME_KEY"); timeKey.Value != "timestamp" {
		t.Errorf("Container env time key is not matched: %v", timeKey)
	}
	if tagPrefix := findEnv(container.Env, "TAG_PREFIX"); tagPrefix.Value != "my-app" {
		t.Errorf("Container env tag prefix is not matched: %v", tagPrefix)
	}
	if aggregatorHost := findEnv(container.Env, "AGGREGATOR_HOST"); aggregatorHost.Value != "my-aggregator.local" {
		t.Errorf("Container env aggregator host is not matched: %v", aggregatorHost)
	}
	if aggregatorPort := findEnv(container.Env, "AGGREGATOR_PORT"); aggregatorPort.Value != "24223" {
		t.Errorf("Container env aggregator port is not matched: %v", aggregatorPort)
	}
	if logFormat := findEnv(container.Env, "LOG_FORMAT"); logFormat.Value != "nginx" {
		t.Errorf("Container env log format is not matched: %v", logFormat)
	}
	if logDir := findEnv(container.Env, "APPLICATION_LOG_DIR"); logDir.Value != "/var/log/nginx" {
		t.Errorf("Container env log dir is not matched: %v", logDir)
	}

	if container.VolumeMounts[0].MountPath != "/var/log/nginx" {
		t.Errorf("Container volume mount path is not matched: %v", container.VolumeMounts[0])
	}

	os.Unsetenv("DOCKER_IMAGE")
	os.Unsetenv("APPLICATION_LOG_DIR")
	os.Unsetenv("AGGREGATOR_HOST")
	os.Unsetenv("AGGREGATOR_PORT")
	os.Unsetenv("TIME_FORMAT")
	os.Unsetenv("TIME_KEY")
	os.Unsetenv("TAG_PREFIX")
	os.Unsetenv("LOG_FORMAT")
}

func findVolume(volumes []corev1.Volume, targetName string) *corev1.Volume {
	for i := range volumes {
		if volumes[i].Name == targetName {
			return &volumes[i]
		}
	}
	return nil
}

func findContainer(containers []corev1.Container, targetName string) *corev1.Container {
	for i := range containers {
		if containers[i].Name == targetName {
			return &containers[i]
		}
	}
	return nil
}

func findEnv(env []corev1.EnvVar, targetName string) *corev1.EnvVar {
	for i := range env {
		if env[i].Name == targetName {
			return &env[i]
		}
	}
	return nil
}
