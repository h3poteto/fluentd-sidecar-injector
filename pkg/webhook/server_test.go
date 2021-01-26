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

	if log := findMount(container.VolumeMounts, VolumeName); log.MountPath != "/var/log/nginx" {
		t.Errorf("Container volume mount path is not matched: %v", log)
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
		t.Errorf("Failed to inject sidecar container: %#v", pod.Spec.Containers)
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

	if log := findMount(container.VolumeMounts, VolumeName); log.MountPath != "/var/log/nginx" {
		t.Errorf("Container volume mount path is not matched: %v", log)
	}
	if config := findMount(container.VolumeMounts, "my-custom-volume"); config.MountPath != "/fluentd/etc/fluent.conf" {
		t.Errorf("Container volume mount custom config is not matched: %#v", config)
	}
}

func TestInjectFluentDWithEnv(t *testing.T) {
	os.Setenv("FLUENTD_DOCKER_IMAGE", "my-fluentd-image:latest")
	os.Setenv("FLUENTD_APPLICATION_LOG_DIR", "/var/log/nginx")
	os.Setenv("FLUENTD_AGGREGATOR_HOST", "my-aggregator.local")
	os.Setenv("FLUENTD_AGGREGATOR_PORT", "24223")
	os.Setenv("FLUENTD_TIME_FORMAT", "%Y/%m/%d %H:%M:%S")
	os.Setenv("FLUENTD_TIME_KEY", "timestamp")
	os.Setenv("FLUENTD_TAG_PREFIX", "my-app")
	os.Setenv("FLUENTD_LOG_FORMAT", "nginx")

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

	if log := findMount(container.VolumeMounts, VolumeName); log.MountPath != "/var/log/nginx" {
		t.Errorf("Container volume mount path is not matched: %v", log)
	}

	os.Unsetenv("FLUENTD_DOCKER_IMAGE")
	os.Unsetenv("FLUENTD_APPLICATION_LOG_DIR")
	os.Unsetenv("FLUENTD_AGGREGATOR_HOST")
	os.Unsetenv("FLUENTD_AGGREGATOR_PORT")
	os.Unsetenv("FLUENTD_TIME_FORMAT")
	os.Unsetenv("FLUENTD_TIME_KEY")
	os.Unsetenv("FLUENTD_TAG_PREFIX")
	os.Unsetenv("FLUENTD_LOG_FORMAT")
}

func TestInjectFluentBit(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "",
			Namespace: "",
			Annotations: map[string]string{
				annotationPrefix + "/injection":           "enabled",
				annotationPrefix + "/collector":           "fluent-bit",
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

	result, err := injectFluentBit(pod)
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
		t.Errorf("Failed to inject sidecar container: %#v", pod.Spec.Containers)
		return
	}
	if container.Image != "ghcr.io/h3poteto/fluentbit-forward:latest" {
		t.Errorf("Container image is not matched: %s", container.Image)
	}

	if refreshInterval := findEnv(container.Env, "REFRESH_INTERVAL"); refreshInterval.Value != "60" {
		t.Errorf("Container env refresh interval is not matched: %v", refreshInterval)
	}
	if rotateWait := findEnv(container.Env, "ROTATE_WAIT"); rotateWait.Value != "5" {
		t.Errorf("Container env rotate wait is not matched: %v", rotateWait)
	}
	if tagPrefix := findEnv(container.Env, "TAG_PREFIX"); tagPrefix.Value != "app" {
		t.Errorf("Container env tag prefix is not matched: %v", tagPrefix)
	}
	if aggregatorHost := findEnv(container.Env, "AGGREGATOR_HOST"); aggregatorHost.Value != "my-aggregator.local" {
		t.Errorf("Container env aggregator host is not matched: %v", aggregatorHost)
	}
	if aggregatorPort := findEnv(container.Env, "AGGREGATOR_PORT"); aggregatorPort.Value != "24224" {
		t.Errorf("Container env aggregator port is not matched: %v", aggregatorPort)
	}
	if logDir := findEnv(container.Env, "APPLICATION_LOG_DIR"); logDir.Value != "/var/log/nginx" {
		t.Errorf("Container env log dir is not matched: %v", logDir)
	}

	if log := findMount(container.VolumeMounts, VolumeName); log.MountPath != "/var/log/nginx" {
		t.Errorf("Container volume mount path is not matched: %v", log)
	}
}

func TestInjectFluentBitWithAnnotations(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "",
			Namespace: "",
			Annotations: map[string]string{
				annotationPrefix + "/injection":           "enabled",
				annotationPrefix + "/collector":           "fluent-bit",
				annotationPrefix + "/docker-image":        "my-fluent-bit-image:latest",
				annotationPrefix + "/aggregator-host":     "my-aggregator.local",
				annotationPrefix + "/application-log-dir": "/var/log/nginx",
				annotationPrefix + "/expose-port":         "80",
				annotationPrefix + "/refresh-interval":    "30",
				annotationPrefix + "/rotate-wait":         "10",
				annotationPrefix + "/config-volume":       "my-custom-config",
				annotationPrefix + "/tag-prefix":          "my-app",
			},
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "my-custom-config",
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

	result, err := injectFluentBit(pod)
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
		t.Errorf("Failed to inject sidecar container: %#v", pod.Spec.Containers)
		return
	}
	if container.Image != "my-fluent-bit-image:latest" {
		t.Errorf("Container image is not matched: %s", container.Image)
	}
	if container.Ports[0].ContainerPort != int32(80) {
		t.Errorf("Container port is not matched: %d", container.Ports[0].ContainerPort)
	}

	if refreshInterval := findEnv(container.Env, "REFRESH_INTERVAL"); refreshInterval.Value != "30" {
		t.Errorf("Container env refresh interval is not matched: %v", refreshInterval)
	}
	if rotateWait := findEnv(container.Env, "ROTATE_WAIT"); rotateWait.Value != "10" {
		t.Errorf("Container env rotate wait is not matched: %v", rotateWait)
	}
	if tagPrefix := findEnv(container.Env, "TAG_PREFIX"); tagPrefix.Value != "my-app" {
		t.Errorf("Container env tag prefix is not matched: %v", tagPrefix)
	}
	if aggregatorHost := findEnv(container.Env, "AGGREGATOR_HOST"); aggregatorHost.Value != "my-aggregator.local" {
		t.Errorf("Container env aggregator host is not matched: %v", aggregatorHost)
	}
	if aggregatorPort := findEnv(container.Env, "AGGREGATOR_PORT"); aggregatorPort.Value != "24224" {
		t.Errorf("Container env aggregator port is not matched: %v", aggregatorPort)
	}
	if logDir := findEnv(container.Env, "APPLICATION_LOG_DIR"); logDir.Value != "/var/log/nginx" {
		t.Errorf("Container env log dir is not matched: %v", logDir)
	}

	if log := findMount(container.VolumeMounts, VolumeName); log.MountPath != "/var/log/nginx" {
		t.Errorf("Container volume mount path is not matched: %v", log)
	}
	if config := findMount(container.VolumeMounts, "my-custom-config"); config.MountPath != "/fluent-bit/etc" {
		t.Errorf("Container volume mount custom config is not matched: %#v", config)
	}
}

func TestInjectFluentBitWithEnv(t *testing.T) {
	os.Setenv("COLLECTOR", "fluent-bit")
	os.Setenv("FLUENTBIT_DOCKER_IMAGE", "my-fluent-bit-image:latest")
	os.Setenv("FLUENTBIT_APPLICATION_LOG_DIR", "/var/log/nginx")
	os.Setenv("FLUENTBIT_TAG_PREFIX", "my-app")
	os.Setenv("FLUENTBIT_AGGREGATOR_HOST", "my-aggregator.local")
	os.Setenv("FLUENTBIT_AGGREGATOR_PORT", "24223")

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

	result, err := injectFluentBit(pod)
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
		t.Errorf("Failed to inject sidecar container: %#v", pod.Spec.Containers)
		return
	}
	if container.Image != "my-fluent-bit-image:latest" {
		t.Errorf("Container image is not matched: %s", container.Image)
	}

	if refreshInterval := findEnv(container.Env, "REFRESH_INTERVAL"); refreshInterval.Value != "60" {
		t.Errorf("Container env refresh interval is not matched: %v", refreshInterval)
	}
	if rotateWait := findEnv(container.Env, "ROTATE_WAIT"); rotateWait.Value != "5" {
		t.Errorf("Container env rotate wait is not matched: %v", rotateWait)
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
	if logDir := findEnv(container.Env, "APPLICATION_LOG_DIR"); logDir.Value != "/var/log/nginx" {
		t.Errorf("Container env log dir is not matched: %v", logDir)
	}

	if log := findMount(container.VolumeMounts, VolumeName); log.MountPath != "/var/log/nginx" {
		t.Errorf("Container volume mount path is not matched: %v", log)
	}

	os.Unsetenv("COLLECTOR")
	os.Unsetenv("FLUENTBIT_DOCKER_IMAGE")
	os.Unsetenv("FLUENTBIT_APPLICATION_LOG_DIR")
	os.Unsetenv("FLUENTBIT_TAG_PREFIX")
	os.Unsetenv("FLUENTBIT_AGGREGATOR_HOST")
	os.Unsetenv("FLUENTBIT_AGGREGATOR_PORT")
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

func findMount(mount []corev1.VolumeMount, targetName string) *corev1.VolumeMount {
	for i := range mount {
		if mount[i].Name == targetName {
			return &mount[i]
		}
	}
	return nil
}
