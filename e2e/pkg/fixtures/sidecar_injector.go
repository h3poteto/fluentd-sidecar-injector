package fixtures

import (
	v1alpha1 "github.com/h3poteto/fluentd-sidecar-injector/pkg/apis/sidecarinjectorcontroller/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewSidecarInjector(collector string) *v1alpha1.SidecarInjector {
	return sidecarInjector(collector)
}

func sidecarInjector(collector string) *v1alpha1.SidecarInjector {
	return &v1alpha1.SidecarInjector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "e2e",
		},
		Spec: v1alpha1.SidecarInjectorSpec{
			Collector: collector,
		},
	}
}
