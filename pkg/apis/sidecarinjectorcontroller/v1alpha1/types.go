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
	// Default collector name which you want to inject. The name must be fluentd or fluent-bit. Default is fluentd.
	Collector string `json:"collector"`
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
