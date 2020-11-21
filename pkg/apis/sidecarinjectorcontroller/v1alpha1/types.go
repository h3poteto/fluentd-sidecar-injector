package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SidecarInjector is a top-level type. A client is created for it.
type SidecarInjector struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              SidecarInjectorSpec `json:"spec"`
	// +optional
	Status SidecarInjectorStatus `json:"status,omitempty"`
}

type SidecarInjectorSpec struct {
	Namespace string `json:"namespace"`
	Collector string `json:"collector"`
}

type SidecarInjectorStatus struct {
	InjectorDeploymentCount int32 `json:"injectorDeploymentCount"`
	InjectorServiceCount    int32 `json:"injectorServiceCount"`
	InjectorServicePort     int32 `json:"injectorServicePort"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SidecarInjectorList
type SidecarInjectorList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SidecarInjector `json:"items"`
}
