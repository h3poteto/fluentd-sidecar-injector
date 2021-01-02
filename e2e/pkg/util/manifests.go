package util

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	serializeryaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"sigs.k8s.io/yaml"
)

// ApplyCRD applies custom resource definitions for sidecar-injector which is located in cmd/config/crd.
func ApplyCRD(ctx context.Context, cfg *rest.Config) error {
	p, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(p, "../config/crd/operator.h3poteto.dev_sidecarinjectors.yaml")

	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return apply(ctx, cfg, buf)
}

// DeleteCRD deletes custom resource definitions for sidecar-injector.
func DeleteCRD(ctx context.Context, cfg *rest.Config) error {
	p, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(p, "../config/crd/operator.h3poteto.dev_sidecarinjectors.yaml")

	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return delete(ctx, cfg, buf)
}

// ApplyRBAC applies role based access control for operator which is located in cmd/config/rbac.
func ApplyRBAC(ctx context.Context, cfg *rest.Config) error {
	p, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(p, "../config/rbac/role.yaml")

	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return apply(ctx, cfg, buf)
}

// DeleteRBAC deletes role based access control for operator.
func DeleteRBAC(ctx context.Context, cfg *rest.Config) error {
	p, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(p, "../config/rbac/role.yaml")

	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return delete(ctx, cfg, buf)
}

func apply(ctx context.Context, cfg *rest.Config, data []byte) error {
	return restAction(ctx, cfg, data, func(ctx context.Context, dr dynamic.ResourceInterface, obj *unstructured.Unstructured, data []byte) error {
		_, err := dr.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
			FieldManager: "e2e",
		})
		return err
	})
}

func delete(ctx context.Context, cfg *rest.Config, data []byte) error {
	return restAction(ctx, cfg, data, func(ctx context.Context, dr dynamic.ResourceInterface, obj *unstructured.Unstructured, data []byte) error {
		return dr.Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
	})
}

type actionFunc func(ctx context.Context, dr dynamic.ResourceInterface, obj *unstructured.Unstructured, data []byte) error

// https://ymmt2005.hatenablog.com/entry/2020/04/14/An_example_of_using_dynamic_client_of_k8s.io/client-go
func restAction(ctx context.Context, cfg *rest.Config, data []byte, action actionFunc) error {
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}

	multidocReader := utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))

	for {
		buf, err := multidocReader.Read()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		var typeMeta runtime.TypeMeta
		if err := yaml.Unmarshal(buf, &typeMeta); err != nil {
			continue
		}
		if typeMeta.Kind == "" {
			continue
		}

		obj := &unstructured.Unstructured{}
		_, gvk, err := serializeryaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(buf, nil, obj)
		if err != nil {
			return err
		}

		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			dr = dyn.Resource(mapping.Resource).Namespace(obj.GetNamespace())
		} else {
			dr = dyn.Resource(mapping.Resource)
		}

		data, err := json.Marshal(obj)
		if err != nil {
			return err
		}

		if err := action(ctx, dr, obj, data); err != nil {
			return err
		}

	}
}
