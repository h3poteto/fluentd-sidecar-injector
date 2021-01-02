package fixtures

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilpointer "k8s.io/utils/pointer"
)

const (
	TestPodLabelKey   = "e2e-test-key"
	TestPodLabelValue = "nginx"
	LogDir            = "/var/log/nginx"
	TestContainerName = "nginx"
)

func NewNginx(ns string) *appsv1.Deployment {
	return nginx(ns)
}

func nginx(ns string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TestContainerName,
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: utilpointer.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					TestPodLabelKey: TestPodLabelValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						TestPodLabelKey: TestPodLabelValue,
					},
					Annotations: map[string]string{
						"fluentd-sidecar-injector.h3poteto.dev/injection":           "enabled",
						"fluentd-sidecar-injector.h3poteto.dev/aggregator-host":     "127.0.0.1",
						"fluentd-sidecar-injector.h3poteto.dev/application-log-dir": LogDir,
						"fluentd-sidecar-injector.h3poteto.dev/time-key":            "time",
						"fluentd-sidecar-injector.h3poteto.dev/tag-prefix":          "test",
					},
				},
				Spec: corev1.PodSpec{
					Volumes:        nil,
					InitContainers: nil,
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
			},
		},
	}
}
