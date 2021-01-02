package util

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

func WaitPodRunning(podList *corev1.PodList) (bool, error) {
	klog.V(4).Infof("Pods are %#v", podList.Items)
	if len(podList.Items) == 0 {
		klog.Info("Pods have not been deployed yet")
		return false, nil
	}
	for i := range podList.Items {
		klog.Infof("Pod %s phase is %s", podList.Items[i].Name, podList.Items[i].Status.Phase)
		if podList.Items[i].Status.Phase != corev1.PodRunning {
			return false, nil
		}
		for _, status := range podList.Items[i].Status.ContainerStatuses {
			if !status.Ready {
				klog.Infof("Container %s in Pod %s is not ready", status.Name, podList.Items[i].Name)
				return false, nil
			}
			if status.State.Running == nil {
				klog.Infof("Container %s in Pod %s is not running", status.Name, podList.Items[i].Name)
				return false, nil
			}
			klog.Infof("Container %s in Pod %s is ready and running", status.Name, podList.Items[i].Name)
		}
	}
	return true, nil
}

func FindContainer(pod *corev1.Pod, containerName string) *corev1.Container {
	for i := range pod.Spec.Containers {
		if pod.Spec.Containers[i].Name == containerName {
			return &pod.Spec.Containers[i]
		}
	}
	return nil
}

func FindMount(mounts []corev1.VolumeMount, volumeName string) *corev1.VolumeMount {
	for i := range mounts {
		if mounts[i].Name == volumeName {
			return &mounts[i]
		}
	}
	return nil
}

func FindVolume(volumes []corev1.Volume, volumeName string) *corev1.Volume {
	for i := range volumes {
		if volumes[i].Name == volumeName {
			return &volumes[i]
		}
	}
	return nil
}
