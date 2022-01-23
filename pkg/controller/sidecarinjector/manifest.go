package sidecarinjector

import (
	"bytes"
	"text/template"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func issuerManifest(issuerName, namespace string, ownerRef *metav1.OwnerReference) (*bytes.Buffer, error) {
	params := map[string]interface{}{
		"IssuerName":      issuerName,
		"Namespace":       namespace,
		"OwnerAPIVersion": ownerRef.APIVersion,
		"OwnerKind":       ownerRef.Kind,
		"OwnerName":       ownerRef.Name,
		"OwnerUID":        ownerRef.UID,
	}
	tpl, err := template.New("issuer").Parse(issuerTmpl)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	if err := tpl.Execute(buf, params); err != nil {
		return nil, err
	}
	return buf, nil
}

func certificateManifest(secretName, certificateName, serviceName, issuerName, namespace string, ownerRef *metav1.OwnerReference) (*bytes.Buffer, error) {
	params := map[string]interface{}{
		"CertificateName": certificateName,
		"Namespace":       namespace,
		"ServiceName":     serviceName,
		"IssuerName":      issuerName,
		"CertSecretName":  secretName,
		"OwnerAPIVersion": ownerRef.APIVersion,
		"OwnerKind":       ownerRef.Kind,
		"OwnerName":       ownerRef.Name,
		"OwnerUID":        ownerRef.UID,
	}
	tpl, err := template.New("certificate").Parse(certificateTmpl)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	if err := tpl.Execute(buf, params); err != nil {
		return nil, err
	}
	return buf, nil
}
