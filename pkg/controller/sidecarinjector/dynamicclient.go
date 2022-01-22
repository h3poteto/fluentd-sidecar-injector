package sidecarinjector

import (
	"context"
	"encoding/json"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type DynamicClient struct {
	client    dynamic.Interface
	discovery discovery.DiscoveryInterface
	mapper    meta.RESTMapper
}

func NewDynamicClient(restConfig *rest.Config, clientset kubernetes.Interface) (*DynamicClient, error) {
	discoveryClient := clientset.Discovery()
	groupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
	dyn, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &DynamicClient{
		client:    dyn,
		discovery: discoveryClient,
		mapper:    mapper,
	}, nil
}

func (d *DynamicClient) ResourceClient(data []byte, obj *unstructured.Unstructured) (dynamic.ResourceInterface, error) {
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, gvk, err := dec.Decode(data, nil, obj)
	if err != nil {
		return nil, err
	}

	mapping, err := d.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}

	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if obj.GetNamespace() == "" {
			obj.SetNamespace(metav1.NamespaceDefault)
		}
		return d.client.Resource(mapping.Resource).Namespace(obj.GetNamespace()), nil
	} else {
		return d.client.Resource(mapping.Resource), nil
	}
}

func (d *DynamicClient) Get(ctx context.Context, client dynamic.ResourceInterface, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return client.Get(ctx, obj.GetName(), metav1.GetOptions{})
}

func (d *DynamicClient) Create(ctx context.Context, client dynamic.ResourceInterface, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return client.Create(ctx, obj, metav1.CreateOptions{})
}

func (d *DynamicClient) Apply(ctx context.Context, client dynamic.ResourceInterface, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	return client.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: "sidecar-injector",
	})
}
