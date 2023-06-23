package webhook

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhlogrus "github.com/slok/kubewebhook/v2/pkg/log/logrus"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var annotationPrefix = "fluentd-sidecar-injector.h3poteto.dev"

const (
	ContainerName = "fluentd-sidecar"
	VolumeName    = "fluentd-sidecar-injector-logs"
)

// GeneralEnv is required environment variables to run this server.
type GeneralEnv struct {
	Collector string `envconfig:"COLLECTOR" default:"fluentd"`
}

// FluentDEnv is required environment variables for fluentd settings.
type FluentDEnv struct {
	DockerImage       string `envconfig:"DOCKER_IMAGE" default:"ghcr.io/h3poteto/fluentd-forward:latest"`
	ApplicationLogDir string `envconfig:"APPLICATION_LOG_DIR"`
	TimeFormat        string `envconfig:"TIME_FORMAT" default:"%Y-%m-%dT%H:%M:%S%z"`
	TimeKey           string `envconfig:"TIME_KEY" default:"time"`
	TagPrefix         string `envconfig:"TAG_PREFIX" default:"app"`
	AggregatorHost    string `envconfig:"AGGREGATOR_HOST"`
	AggregatorPort    string `envconfig:"AGGREGATOR_PORT" default:"24224"`
	LogFormat         string `envconfig:"LOG_FORMAT" default:"json"`
	CustomEnv         string `envconfig:"CUSTOM_ENV"`
}

type FluentBitEnv struct {
	DockerImage       string `envconfig:"DOCKER_IMAGE" default:"ghcr.io/h3poteto/fluentbit-forward:latest"`
	ApplicationLogDir string `envconfig:"APPLICATION_LOG_DIR"`
	TagPrefix         string `envconfig:"TAG_PREFIX" default:"app"`
	AggregatorHost    string `envconfig:"AGGREGATOR_HOST"`
	AggregatorPort    string `envconfig:"AGGREGATOR_PORT" default:"24224"`
	CustomEnv         string `envconfig:"CUSTOM_ENV"`
}

var logger kwhlog.Logger

// StartServer run webhook server.
func StartServer(tlsCertFile, tlsKeyFile string) error {
	logrusLogEntry := logrus.NewEntry(logrus.New())
	logrusLogEntry.Logger.SetLevel(logrus.DebugLevel)
	logger = kwhlogrus.NewLogrus(logrusLogEntry)

	mutator := kwhmutating.MutatorFunc(sidecarInjectMutator)

	config := kwhmutating.WebhookConfig{
		ID:      "fluentdSidecarInjector",
		Obj:     &corev1.Pod{},
		Mutator: mutator,
		Logger:  logger,
	}
	webhook, err := kwhmutating.NewWebhook(config)
	if err != nil {
		return fmt.Errorf("Failed to create webhook: %s", err)
	}

	handler, err := kwhhttp.HandlerFor(kwhhttp.HandlerConfig{Webhook: webhook, Logger: logger})
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

// sidecarInjectMutator mutates requested pod definition to inject fluentd as sidecar.
// This function retunrs bool, and error to detect stop applying.
// If return false, API server does not stop applying. But if return true, API server stop applying, and say errors to kubectl.
func sidecarInjectMutator(_ context.Context, _ *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhmutating.MutatorResult, error) {
	logger.Debugf("Receive request")

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return &kwhmutating.MutatorResult{}, nil
	}
	logger.Debugf("Receive pod: %#v", pod)

	if pod.Annotations[annotationPrefix+"/injection"] != "enabled" {
		logger.Debugf("Skip injector because annotation is not specified")
		return &kwhmutating.MutatorResult{}, nil
	}

	var generalEnv GeneralEnv
	err := envconfig.Process("", &generalEnv)
	if err != nil {
		return &kwhmutating.MutatorResult{}, err
	}

	collector := generalEnv.Collector
	if value, ok := pod.Annotations[annotationPrefix+"/collector"]; ok {
		collector = value
	}
	switch collector {
	case "fluentd", "":
		return injectFluentD(pod)
	case "fluent-bit":
		return injectFluentBit(pod)
	default:
		return &kwhmutating.MutatorResult{}, fmt.Errorf("collector must be fluentd or fluent-bit, %s is not matched", collector)
	}
}

func injectFluentD(pod *corev1.Pod) (*kwhmutating.MutatorResult, error) {
	var fluentdEnv FluentDEnv
	err := envconfig.Process("fluentd", &fluentdEnv)
	if err != nil {
		return &kwhmutating.MutatorResult{}, err
	}

	dockerImage := fluentdEnv.DockerImage
	if value, ok := pod.Annotations[annotationPrefix+"/docker-image"]; ok {
		dockerImage = value
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name: VolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	resourceRequirements := corev1.ResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceMemory: *resource.NewQuantity(200*1024*1024, resource.BinarySI),
			corev1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
		},
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceMemory: *resource.NewQuantity(1000*1024*1024, resource.BinarySI),
		},
	}

	if value, ok := pod.Annotations[annotationPrefix+"/memory-request"]; ok {
		resourceRequirements.Requests[corev1.ResourceMemory] = resource.Quantity{
			Format: resource.Format(value),
		}
	}

	if value, ok := pod.Annotations[annotationPrefix+"/memory-limit"]; ok {
		resourceRequirements.Limits[corev1.ResourceMemory] = resource.Quantity{
			Format: resource.Format(value),
		}
	}

	if value, ok := pod.Annotations[annotationPrefix+"/cpu-request"]; ok {
		resourceRequirements.Requests[corev1.ResourceCPU] = resource.Quantity{
			Format: resource.Format(value),
		}
	}

	if value, ok := pod.Annotations[annotationPrefix+"/cpu-limit"]; ok {
		resourceRequirements.Limits[corev1.ResourceCPU] = resource.Quantity{
			Format: resource.Format(value),
		}
	}

	sidecar := corev1.Container{
		Name:      ContainerName,
		Image:     dockerImage,
		Resources: resourceRequirements,
	}

	if value, ok := pod.Annotations[annotationPrefix+"/expose-port"]; ok {
		port, _ := strconv.Atoi(value)
		sidecar.Ports = []corev1.ContainerPort{{ContainerPort: int32(port)}}
	}

	// Override env with Pod's annotations.
	sendTimeout := "60s"
	if value, ok := pod.Annotations[annotationPrefix+"/send-timeout"]; ok {
		sendTimeout = value
	}
	sidecar.Env = append(sidecar.Env, corev1.EnvVar{
		Name:  "SEND_TIMEOUT",
		Value: sendTimeout,
	})

	recoverWait := "10s"
	if value, ok := pod.Annotations[annotationPrefix+"/recover-wait"]; ok {
		recoverWait = value
	}
	sidecar.Env = append(sidecar.Env, corev1.EnvVar{
		Name:  "RECOVER_WAIT",
		Value: recoverWait,
	})

	hardTimeout := "120s"
	if value, ok := pod.Annotations[annotationPrefix+"/hard-timeout"]; ok {
		hardTimeout = value
	}
	sidecar.Env = append(sidecar.Env, corev1.EnvVar{
		Name:  "HARD_TIMEOUT",
		Value: hardTimeout,
	})

	// Override env with fluentdEnv and Pod's annotations.
	aggregatorHost := fluentdEnv.AggregatorHost
	if value, ok := pod.Annotations[annotationPrefix+"/aggregator-host"]; ok {
		aggregatorHost = value
	}

	if aggregatorHost == "" {
		return &kwhmutating.MutatorResult{}, errors.New("aggregator host is required")
	}
	sidecar.Env = append(sidecar.Env, corev1.EnvVar{
		Name:  "AGGREGATOR_HOST",
		Value: aggregatorHost,
	})

	aggregatorPort := fluentdEnv.AggregatorPort
	if value, ok := pod.Annotations[annotationPrefix+"/aggregator-port"]; ok {
		aggregatorPort = value
	}

	if aggregatorPort != "" {
		sidecar.Env = append(sidecar.Env, corev1.EnvVar{
			Name:  "AGGREGATOR_PORT",
			Value: aggregatorPort,
		})
	}

	logFormat := fluentdEnv.LogFormat
	if value, ok := pod.Annotations[annotationPrefix+"/log-format"]; ok {
		logFormat = value
	}

	if logFormat != "" {
		sidecar.Env = append(sidecar.Env, corev1.EnvVar{
			Name:  "LOG_FORMAT",
			Value: logFormat,
		})
	}

	customEnv := fluentdEnv.CustomEnv
	if value, ok := pod.Annotations[annotationPrefix+"/custom-env"]; ok {
		customEnv = value
	}

	if customEnv != "" {
		sidecar.Env = append(sidecar.Env, corev1.EnvVar{
			Name:  "CUSTOM_ENV",
			Value: customEnv,
		})
	}

	applicationLogDir := fluentdEnv.ApplicationLogDir
	if value, ok := pod.Annotations[annotationPrefix+"/application-log-dir"]; ok {
		applicationLogDir = value
	}
	if applicationLogDir == "" {
		return &kwhmutating.MutatorResult{}, errors.New("application log dir is required")
	}
	sidecar.Env = append(sidecar.Env, corev1.EnvVar{
		Name:  "APPLICATION_LOG_DIR",
		Value: applicationLogDir,
	})

	volumeMount := corev1.VolumeMount{
		Name:      VolumeName,
		ReadOnly:  false,
		MountPath: applicationLogDir,
	}
	sidecar.VolumeMounts = []corev1.VolumeMount{
		volumeMount,
	}

	mountsCnt := len(sidecar.VolumeMounts)
	if value, ok := pod.Annotations[annotationPrefix+"/config-volume"]; ok {
		volumes := pod.Spec.Volumes
		for i := range volumes {
			if name := volumes[i].Name; name == value {
				sidecar.VolumeMounts = append(sidecar.VolumeMounts, corev1.VolumeMount{
					Name:      name,
					MountPath: "/fluentd/etc"})
				break
			}
		}

		if mountsCnt == len(sidecar.VolumeMounts) {
			return &kwhmutating.MutatorResult{}, errors.New("config volume does not exist")
		}
	}

	tagPrefix := fluentdEnv.TagPrefix
	if value, ok := pod.Annotations[annotationPrefix+"/tag-prefix"]; ok {
		tagPrefix = value
	}
	if tagPrefix != "" {
		sidecar.Env = append(sidecar.Env, corev1.EnvVar{
			Name:  "TAG_PREFIX",
			Value: tagPrefix,
		})
	}

	timeKey := fluentdEnv.TimeKey
	if value, ok := pod.Annotations[annotationPrefix+"/time-key"]; ok {
		timeKey = value
	}
	if timeKey != "" {
		sidecar.Env = append(sidecar.Env, corev1.EnvVar{
			Name:  "TIME_KEY",
			Value: timeKey,
		})
	}

	// Add Downward API
	// ref: https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/#the-downward-api
	sidecar.Env = append(sidecar.Env,
		corev1.EnvVar{
			Name: "NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
		corev1.EnvVar{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		corev1.EnvVar{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		corev1.EnvVar{
			Name: "POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
		corev1.EnvVar{
			Name: "POD_SERVICE_ACCOUNT",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.serviceAccountName",
				},
			},
		},
		corev1.EnvVar{
			Name: "CPU_REQUEST",
			ValueFrom: &corev1.EnvVarSource{
				ResourceFieldRef: &corev1.ResourceFieldSelector{
					ContainerName: ContainerName,
					Resource:      "requests.cpu",
				},
			},
		},
		corev1.EnvVar{
			Name: "CPU_LIMIT",
			ValueFrom: &corev1.EnvVarSource{
				ResourceFieldRef: &corev1.ResourceFieldSelector{
					ContainerName: ContainerName,
					Resource:      "limits.cpu",
				},
			},
		},
		corev1.EnvVar{
			Name: "MEM_REQUEST",
			ValueFrom: &corev1.EnvVarSource{
				ResourceFieldRef: &corev1.ResourceFieldSelector{
					ContainerName: ContainerName,
					Resource:      "requests.memory",
				},
			},
		},
		corev1.EnvVar{
			Name: "MEM_LIMIT",
			ValueFrom: &corev1.EnvVarSource{
				ResourceFieldRef: &corev1.ResourceFieldSelector{
					ContainerName: ContainerName,
					Resource:      "limits.memory",
				},
			},
		},
	)

	timeFormat := fluentdEnv.TimeFormat
	if value, ok := pod.Annotations[annotationPrefix+"/time-format"]; ok {
		timeFormat = value
	}
	if timeFormat != "" {
		sidecar.Env = append(sidecar.Env, corev1.EnvVar{
			Name:  "TIME_FORMAT",
			Value: timeFormat,
		})
	}

	// Inject volume mount for all containers in the pod.
	var containers []corev1.Container

	for _, container := range pod.Spec.Containers {
		container.VolumeMounts = append(container.VolumeMounts, volumeMount)
		containers = append(containers, container)
	}
	containers = append(containers, sidecar)

	pod.Spec.Containers = containers

	return &kwhmutating.MutatorResult{
		MutatedObject: pod,
	}, nil
}

func injectFluentBit(pod *corev1.Pod) (*kwhmutating.MutatorResult, error) {
	var fluentBitEnv FluentBitEnv
	err := envconfig.Process("fluentbit", &fluentBitEnv)
	if err != nil {
		return &kwhmutating.MutatorResult{}, err
	}

	dockerImage := fluentBitEnv.DockerImage
	if value, ok := pod.Annotations[annotationPrefix+"/docker-image"]; ok {
		dockerImage = value
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name: VolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	sidecar := corev1.Container{
		Name:  ContainerName,
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
	}

	if value, ok := pod.Annotations[annotationPrefix+"/expose-port"]; ok {
		port, _ := strconv.Atoi(value)
		sidecar.Ports = []corev1.ContainerPort{{ContainerPort: int32(port)}}
	}

	// Override env with Pod's annotations.
	refreshInterval := "60"
	if value, ok := pod.Annotations[annotationPrefix+"/refresh-interval"]; ok {
		refreshInterval = value
	}
	sidecar.Env = append(sidecar.Env, corev1.EnvVar{
		Name:  "REFRESH_INTERVAL",
		Value: refreshInterval,
	})

	rotateWait := "5"
	if value, ok := pod.Annotations[annotationPrefix+"/rotate-wait"]; ok {
		rotateWait = value
	}
	sidecar.Env = append(sidecar.Env, corev1.EnvVar{
		Name:  "ROTATE_WAIT",
		Value: rotateWait,
	})

	// Override env with fluentBitEnv and Pod's annotations.
	aggregatorHost := fluentBitEnv.AggregatorHost
	if value, ok := pod.Annotations[annotationPrefix+"/aggregator-host"]; ok {
		aggregatorHost = value
	}

	if aggregatorHost == "" {
		return &kwhmutating.MutatorResult{}, errors.New("aggregator host is required")
	}
	sidecar.Env = append(sidecar.Env, corev1.EnvVar{
		Name:  "AGGREGATOR_HOST",
		Value: aggregatorHost,
	})

	aggregatorPort := fluentBitEnv.AggregatorPort
	if value, ok := pod.Annotations[annotationPrefix+"/aggregator-port"]; ok {
		aggregatorPort = value
	}

	if aggregatorPort != "" {
		sidecar.Env = append(sidecar.Env, corev1.EnvVar{
			Name:  "AGGREGATOR_PORT",
			Value: aggregatorPort,
		})
	}

	customEnv := fluentBitEnv.CustomEnv
	if value, ok := pod.Annotations[annotationPrefix+"/custom-env"]; ok {
		customEnv = value
	}

	if customEnv != "" {
		sidecar.Env = append(sidecar.Env, corev1.EnvVar{
			Name:  "CUSTOM_ENV",
			Value: customEnv,
		})
	}

	applicationLogDir := fluentBitEnv.ApplicationLogDir
	if value, ok := pod.Annotations[annotationPrefix+"/application-log-dir"]; ok {
		applicationLogDir = value
	}
	if applicationLogDir == "" {
		return &kwhmutating.MutatorResult{}, errors.New("application log dir is required")
	}
	sidecar.Env = append(sidecar.Env, corev1.EnvVar{
		Name:  "APPLICATION_LOG_DIR",
		Value: applicationLogDir,
	})

	volumeMount := corev1.VolumeMount{
		Name:      VolumeName,
		ReadOnly:  false,
		MountPath: applicationLogDir,
	}
	sidecar.VolumeMounts = []corev1.VolumeMount{
		volumeMount,
	}

	mountsCnt := len(sidecar.VolumeMounts)
	if value, ok := pod.Annotations[annotationPrefix+"/config-volume"]; ok {
		volumes := pod.Spec.Volumes
		for i := range volumes {
			if name := volumes[i].Name; name == value {
				sidecar.VolumeMounts = append(sidecar.VolumeMounts, corev1.VolumeMount{
					Name:      name,
					MountPath: "/fluent-bit/etc"})
				break
			}
		}

		if mountsCnt == len(sidecar.VolumeMounts) {
			return &kwhmutating.MutatorResult{}, errors.New("config volume does not exist")
		}
	}

	tagPrefix := fluentBitEnv.TagPrefix
	if value, ok := pod.Annotations[annotationPrefix+"/tag-prefix"]; ok {
		tagPrefix = value
	}
	if tagPrefix != "" {
		sidecar.Env = append(sidecar.Env, corev1.EnvVar{
			Name:  "TAG_PREFIX",
			Value: tagPrefix,
		})
	}

	// Add Downward API
	// ref: https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/#the-downward-api
	sidecar.Env = append(sidecar.Env,
		corev1.EnvVar{
			Name: "NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
		corev1.EnvVar{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		corev1.EnvVar{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		corev1.EnvVar{
			Name: "POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
		corev1.EnvVar{
			Name: "POD_SERVICE_ACCOUNT",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.serviceAccountName",
				},
			},
		},
		corev1.EnvVar{
			Name: "CPU_REQUEST",
			ValueFrom: &corev1.EnvVarSource{
				ResourceFieldRef: &corev1.ResourceFieldSelector{
					ContainerName: ContainerName,
					Resource:      "requests.cpu",
				},
			},
		},
		corev1.EnvVar{
			Name: "CPU_LIMIT",
			ValueFrom: &corev1.EnvVarSource{
				ResourceFieldRef: &corev1.ResourceFieldSelector{
					ContainerName: ContainerName,
					Resource:      "limits.cpu",
				},
			},
		},
		corev1.EnvVar{
			Name: "MEM_REQUEST",
			ValueFrom: &corev1.EnvVarSource{
				ResourceFieldRef: &corev1.ResourceFieldSelector{
					ContainerName: ContainerName,
					Resource:      "requests.memory",
				},
			},
		},
		corev1.EnvVar{
			Name: "MEM_LIMIT",
			ValueFrom: &corev1.EnvVarSource{
				ResourceFieldRef: &corev1.ResourceFieldSelector{
					ContainerName: ContainerName,
					Resource:      "limits.memory",
				},
			},
		},
	)

	// Inject volume mount for all containers in the pod.
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		container.VolumeMounts = append(container.VolumeMounts, volumeMount)
	}
	pod.Spec.Containers = append(pod.Spec.Containers, sidecar)

	return &kwhmutating.MutatorResult{
		MutatedObject: pod,
	}, nil
}
