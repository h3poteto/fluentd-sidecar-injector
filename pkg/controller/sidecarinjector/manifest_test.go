package sidecarinjector

import (
	_ "embed"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:embed testdata/issuer.yaml
var testIssuer string

//go:embed testdata/certificate.yaml
var testCertificate string

func TestIssuerManifest(t *testing.T) {
	issuerName := "selfsigned"
	namespace := "sandbox"
	ownerRef := metav1.OwnerReference{
		APIVersion: "operator.h3poteto.dev/v1alpha1",
		Kind:       "SidecarInjector",
		Name:       "my-injector",
		UID:        "sample-uid",
	}
	manifest, err := issuerManifest(issuerName, namespace, &ownerRef)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if manifest.String() != testIssuer {
		t.Errorf("Manifest does not match: expected: %s, actual: %s", testIssuer, manifest.String())
	}
}

func TestCertificateManifest(t *testing.T) {
	certificateName := "my-certificate"
	namespace := "sandbox"
	serviceName := "webhook-injector"
	issuerName := "selfsigned"
	certSecretName := "serving-cert"
	ownerRef := metav1.OwnerReference{
		APIVersion: "operator.h3poteto.dev/v1alpha1",
		Kind:       "SidecarInjector",
		Name:       "my-injector",
		UID:        "sample-uid",
	}
	manifest, err := certificateManifest(certSecretName, certificateName, serviceName, issuerName, namespace, &ownerRef)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if manifest.String() != testCertificate {
		t.Errorf("Manifest does not match: expected: %s, actual: %s", testCertificate, manifest.String())
	}
}
