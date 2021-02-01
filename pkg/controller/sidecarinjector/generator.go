package sidecarinjector

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	sidecarinjectorv1alpha1 "github.com/h3poteto/fluentd-sidecar-injector/pkg/apis/sidecarinjectorcontroller/v1alpha1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilpointer "k8s.io/utils/pointer"
)

const (
	serverKeyName           = "key.pem"
	serverCertName          = "cert.pem"
	WebhookServerLabelKey   = "sidecarinjectors.operator.h3poteto.dev"
	WebhookServerLabelValue = "webhook-pod"
)

func newDeployment(sidecarInjector *sidecarinjectorv1alpha1.SidecarInjector, namespace, secretName, image string) *appsv1.Deployment {
	env := []corev1.EnvVar{
		{
			Name:  "COLLECTOR",
			Value: sidecarInjector.Spec.Collector,
		},
	}
	if sidecarInjector.Spec.FluentD != nil {
		if sidecarInjector.Spec.FluentD.DockerImage != "" {
			env = append(env, corev1.EnvVar{
				Name:  "FLUENTD_DOCKER_IMAGE",
				Value: sidecarInjector.Spec.FluentD.DockerImage,
			})
		}
		if sidecarInjector.Spec.FluentD.AggregatorHost != "" {
			env = append(env, corev1.EnvVar{
				Name:  "FLUENTD_AGGREGATOR_HOST",
				Value: sidecarInjector.Spec.FluentD.AggregatorHost,
			})
		}
		if sidecarInjector.Spec.FluentD.AggregatorPort != 0 {
			env = append(env, corev1.EnvVar{
				Name:  "FLUENTD_AGGREGATOR_PORT",
				Value: fmt.Sprintf("%d", sidecarInjector.Spec.FluentD.AggregatorPort),
			})
		}
		if sidecarInjector.Spec.FluentD.ApplicationLogDir != "" {
			env = append(env, corev1.EnvVar{
				Name:  "FLUENTD_APPLICATION_LOG_DIR",
				Value: sidecarInjector.Spec.FluentD.ApplicationLogDir,
			})
		}
		if sidecarInjector.Spec.FluentD.TagPrefix != "" {
			env = append(env, corev1.EnvVar{
				Name:  "FLUENTD_TAG_PREFIX",
				Value: sidecarInjector.Spec.FluentD.TagPrefix,
			})
		}
		if sidecarInjector.Spec.FluentD.TimeKey != "" {
			env = append(env, corev1.EnvVar{
				Name:  "FLUENTD_TIME_KEY",
				Value: sidecarInjector.Spec.FluentD.TimeKey,
			})
		}
		if sidecarInjector.Spec.FluentD.TimeFormat != "" {
			env = append(env, corev1.EnvVar{
				Name:  "FLUENTD_TIME_FORMAT",
				Value: sidecarInjector.Spec.FluentD.TimeFormat,
			})
		}
		if sidecarInjector.Spec.FluentD.CustomEnv != "" {
			env = append(env, corev1.EnvVar{
				Name:  "FLUENTD_CUSTOM_ENV",
				Value: sidecarInjector.Spec.FluentD.CustomEnv,
			})
		}
	}
	if sidecarInjector.Spec.FluentBit != nil {
		if sidecarInjector.Spec.FluentBit.DockerImage != "" {
			env = append(env, corev1.EnvVar{
				Name:  "FLUENTBIT_DOCKER_IMAGE",
				Value: sidecarInjector.Spec.FluentBit.DockerImage,
			})
		}
		if sidecarInjector.Spec.FluentBit.AggregatorHost != "" {
			env = append(env, corev1.EnvVar{
				Name:  "FLUENTBIT_AGGREGATOR_HOST",
				Value: sidecarInjector.Spec.FluentBit.AggregatorHost,
			})
		}
		if sidecarInjector.Spec.FluentBit.AggregatorPort != 0 {
			env = append(env, corev1.EnvVar{
				Name:  "FLUENTBIT_AGGREGATOR_PORT",
				Value: fmt.Sprintf("%d", sidecarInjector.Spec.FluentBit.AggregatorPort),
			})
		}
		if sidecarInjector.Spec.FluentBit.ApplicationLogDir != "" {
			env = append(env, corev1.EnvVar{
				Name:  "FLUENTBIT_APPLICATION_LOG_DIR",
				Value: sidecarInjector.Spec.FluentBit.ApplicationLogDir,
			})
		}
		if sidecarInjector.Spec.FluentBit.TagPrefix != "" {
			env = append(env, corev1.EnvVar{
				Name:  "FLUENTBIT_TAG_PREFIX",
				Value: sidecarInjector.Spec.FluentBit.TagPrefix,
			})
		}
		if sidecarInjector.Spec.FluentBit.CustomEnv != "" {
			env = append(env, corev1.EnvVar{
				Name:  "FLUENTBIT_CUSTOM_ENV",
				Value: sidecarInjector.Spec.FluentBit.CustomEnv,
			})
		}
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sidecarInjector.Name + "-handler",
			Namespace: namespace,
			Labels: map[string]string{
				WebhookServerLabelKey: "webhook-deployment",
			},
			Annotations: map[string]string{},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(sidecarInjector, schema.GroupVersionKind{
					Group:   sidecarinjectorv1alpha1.SchemeGroupVersion.Group,
					Version: sidecarinjectorv1alpha1.SchemeGroupVersion.Version,
					Kind:    "SidecarInjector",
				}),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: utilpointer.Int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					WebhookServerLabelKey: WebhookServerLabelValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						WebhookServerLabelKey: WebhookServerLabelValue,
					},
					Annotations: map[string]string{},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "webhook-certs",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: secretName,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:    "webhook-handler",
							Image:   image,
							Command: nil,
							Args: []string{
								"/fluentd-sidecar-injector",
								"webhook",
								"--tls-cert-file=/etc/webhook/certs/" + serverCertName,
								"--tls-key-file=/etc/webhook/certs/" + serverKeyName,
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "https",
									ContainerPort: 8080,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							EnvFrom: nil,
							Env:     env,
							Resources: corev1.ResourceRequirements{
								Limits: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceMemory: {
										Format: resource.Format("500Mi"),
									},
									corev1.ResourceCPU: {
										Format: resource.Format("1000m"),
									},
								},
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceMemory: {
										Format: resource.Format("200Mi"),
									},
									corev1.ResourceCPU: {
										Format: resource.Format("100m"),
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "webhook-certs",
									ReadOnly:  true,
									MountPath: "/etc/webhook/certs",
								},
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromInt(8080),
									},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      60,
								PeriodSeconds:       20,
								SuccessThreshold:    1,
								FailureThreshold:    4,
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromInt(8080),
									},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      60,
								PeriodSeconds:       10,
								SuccessThreshold:    2,
								FailureThreshold:    2,
							},
							ImagePullPolicy: corev1.PullAlways,
						},
					},
					ServiceAccountName: "default",
				},
			},
		},
	}
	return deployment
}

func newSecret(sidecarInjector *sidecarinjectorv1alpha1.SidecarInjector, namespace, serviceName, secretName string) (*corev1.Secret, []byte, error) {
	key, cert, err := NewCertificates(serviceName, namespace)
	if err != nil {
		return nil, nil, err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"sidecarinjectors.operator.h3poteto.dev": "webhook-certs",
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(sidecarInjector, schema.GroupVersionKind{
					Group:   sidecarinjectorv1alpha1.SchemeGroupVersion.Group,
					Version: sidecarinjectorv1alpha1.SchemeGroupVersion.Version,
					Kind:    "SidecarInjector",
				}),
			},
		},
		Data: map[string][]byte{
			serverKeyName:  key,
			serverCertName: cert,
		},
		Type: corev1.SecretTypeOpaque,
	}
	return secret, cert, nil
}

func NewCertificates(serviceName, namespace string) ([]byte, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	name := pkix.Name{
		Country:      []string{},
		Organization: []string{},
		Locality:     []string{},
		CommonName:   serviceName + "." + namespace + ".svc",
	}

	CA := x509.Certificate{
		SerialNumber:          big.NewInt(2048),
		Subject:               name,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		// To generate extensions included SANs.
		DNSNames: []string{
			serviceName + "." + namespace + ".svc",
		},
	}
	cert, err := x509.CreateCertificate(rand.Reader, &CA, &CA, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}
	keyPem := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	certPem := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	}

	return pem.EncodeToMemory(keyPem), pem.EncodeToMemory(certPem), nil
}

func newService(sidecarInjector *sidecarinjectorv1alpha1.SidecarInjector, namespace, serviceName string) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
			Labels: map[string]string{
				"sidecarinjectors.operator.h3poteto.dev": "webhook-service",
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(sidecarInjector, schema.GroupVersionKind{
					Group:   sidecarinjectorv1alpha1.SchemeGroupVersion.Group,
					Version: sidecarinjectorv1alpha1.SchemeGroupVersion.Version,
					Kind:    "SidecarInjector",
				}),
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "https",
					Protocol:   corev1.ProtocolTCP,
					Port:       443,
					TargetPort: intstr.FromInt(8080),
				},
			},
			Selector: map[string]string{
				"sidecarinjectors.operator.h3poteto.dev": "webhook-pod",
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
	return service
}

func newMutatingWebhookConfiguration(sidecarInjector *sidecarinjectorv1alpha1.SidecarInjector, mutatingName, serviceNamespace, serviceName string, serverCertificate []byte) *admissionregistrationv1.MutatingWebhookConfiguration {
	ignore := admissionregistrationv1.Ignore
	allscopes := admissionregistrationv1.AllScopes
	equivalent := admissionregistrationv1.Equivalent
	sideeffect := admissionregistrationv1.SideEffectClassNone
	mutating := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: mutatingName,
			Labels: map[string]string{
				"sidecarinjectors.operator.h3poteto.dev": "webhook-configuration",
				"kind":                                   "mutator",
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(sidecarInjector, schema.GroupVersionKind{
					Group:   sidecarinjectorv1alpha1.SchemeGroupVersion.Group,
					Version: sidecarinjectorv1alpha1.SchemeGroupVersion.Version,
					Kind:    "SidecarInjector",
				}),
			},
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: serviceName + "." + serviceNamespace + ".svc",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: serviceNamespace,
						Name:      serviceName,
						Path:      utilpointer.StringPtr("/mutate"),
					},
					CABundle: serverCertificate,
				},
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1.OperationType{
							"CREATE",
						},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
							Scope:       &allscopes,
						},
					},
				},
				FailurePolicy: &ignore,
				MatchPolicy:   &equivalent,
				ObjectSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "sidecarinjectors.operator.h3poteto.dev",
							Operator: metav1.LabelSelectorOpNotIn,
							Values:   []string{"webhook-pod"},
						},
					},
				},
				SideEffects:             &sideeffect,
				TimeoutSeconds:          utilpointer.Int32Ptr(30),
				AdmissionReviewVersions: []string{"v1beta1"},
			},
		},
	}

	return mutating
}
