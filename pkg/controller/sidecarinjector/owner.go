package sidecarinjector

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// GetOwnerDeployment gets a deployment which owns the pod.
func (c *Controller) GetOwnerDeployment(ctx context.Context, ns, name string) (*appsv1.Deployment, error) {
	pod, err := c.kubeclientset.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get pod: %v", err)
		return nil, err
	}

	ownerRS := findOwner(pod.OwnerReferences, "ReplicaSet")
	if ownerRS == nil {
		return nil, fmt.Errorf("failed to get OwnerReferences in Pod %s/%s", pod.Namespace, pod.Name)
	}

	rs, err := c.kubeclientset.AppsV1().ReplicaSets(ns).Get(ctx, ownerRS.Name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get replicaset: %v", err)
		return nil, err
	}

	ownerDeploy := findOwner(rs.OwnerReferences, "Deployment")
	if ownerDeploy == nil {
		return nil, fmt.Errorf("failed to get OwnerReferences in ReplicaSet %s/%s", rs.Namespace, rs.Name)
	}

	deploy, err := c.kubeclientset.AppsV1().Deployments(ns).Get(ctx, ownerDeploy.Name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get deployment: %v", err)
		return nil, err
	}

	return deploy, nil
}

func findOwner(refs []metav1.OwnerReference, kind string) *metav1.OwnerReference {
	for i := range refs {
		o := &refs[i]
		if o.Kind == kind {
			return o
		}
	}
	return nil
}
