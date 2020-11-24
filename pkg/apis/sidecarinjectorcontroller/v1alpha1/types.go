package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// SidecarInjector is a top-level type. A client is created for it.
type SidecarInjector struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              SidecarInjectorSpec   `json:"spec"`
	Status            SidecarInjectorStatus `json:"status,omitempty"`
}

// SidecarInjectorSpec defines the desired state of SidecarInjector
type SidecarInjectorSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type:=string
	// +kubebuilder:default=fluentd
	// +kubebuilder:validation:Enum=fluentd;fluent-bit
	// Default collector name which you want to inject. The name must be fluentd or fluent-bit. Default is fluentd.
	Collector string `json:"collector"`
	// +optional
	// +nullable
	// Please specify this argument when you specify fluentd as collector.
	FluentD *FluentDSpec `json:"fluentd"`
	// +optional
	// +nullable
	// Please specify this argument when you specify fluent-bit as collector
	FluentBit *FluentBitSpec `json:"fluentbit"`
}

// SdecarInjectorStatus defines the observed state of SidecarInjector
type SidecarInjectorStatus struct {
	InjectorDeploymentName string `json:"injectorDeploymentName"`
	// Available pods count under the deployment of SidecarInjector.
	InjectorPodCount int32 `json:"injectorPodCount"`
	// Whether the webhook service is available.
	InjectorServiceReady bool `json:"injectorServiceReady"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// SidecarInjectorList
type SidecarInjectorList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SidecarInjector `json:"items"`
}

// FluentDSpec describe fluentd options for SidcarInjector.
type FluentDSpec struct {
	// +optional
	// Docker image name which you want to inject to your pods as sidecars. For example, ghcr.io/h3poteto/fluentd-forward:latest
	DockerImage string `json:"dockerImage"`
	// +optional
	// A FluentD hostname as a aggregator. Injected fluentd pods will send logs to this endpoint.
	AggregatorHost string `json:"aggregatorHost"`
	// +optional
	// A FluentD port number as a aggregator.
	AggregatorPort int32 `json:"aggregatorPort"`
	// +optional
	// Lod directory path in your pods. SidecarInjector will mount a volume in this directory, and share it with injected fluentd pod. So fluentd pod can read application logs in this volume.
	ApplicationLogDir string `json:"applicationLogDir"`
	// +optional
	// This tag is prefix of received log's tag. Injected fluentd will add this prefix for all log's tag.
	TagPrefix string `json:"tagPrefix"`
	// +optional
	// A option for fluentd configuration, time_key.
	TimeKey string `json:"timeKey"`
	// +optional
	// A option for fluentd configuration, time_format.
	TimeFormat string `json:"timeFormat"`
	// +optional
	// Additional environment variables for SidecarInjector
	CustomEnv string `json:"customEnv"`
}

// FluentBitSpec describe fluent-bit options for SidecarInjector.
type FluentBitSpec struct {
	// +optional
	// Docker image name which you want to inject to your pods as sidecars. For example, ghcr.io/h3poteto/fluentbit-forward:latest
	DockerImage string `json:"dockerImage"`
	// +optional
	// A FluentD hostname as a aggregator. Injected fluent-bit pods will send logs to this endpoint.
	AggregatorHost string `json:"aggregatorHost"`
	// +optional
	// A FluentD port number as a aggregator.
	AggregatorPort int32 `json:"aggregatorPort"`
	// +optional
	// Lod directory path in your pods. SidecarInjector will mount a volume in this directory, and share it with injected fluent-bit pod. So fluent-bit pod can read application logs in this volume.
	ApplicationLogDir string `json:"applicationLogDir"`
	// +optional
	// This tag is prefix of received log's tag. Injected fluent-bit will add this prefix for all log's tag.
	TagPrefix string `json:"tagPrefix"`
	// +optional
	// Additional environment variables for SidecarInjector
	CustomEnv string `json:"customEnv"`
}
