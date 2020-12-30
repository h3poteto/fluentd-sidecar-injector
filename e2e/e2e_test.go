package e2e_test

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/h3poteto/fluentd-sidecar-injector/e2e/pkg/fixtures"
	"github.com/h3poteto/fluentd-sidecar-injector/e2e/pkg/util"
	clientset "github.com/h3poteto/fluentd-sidecar-injector/pkg/client/clientset/versioned"
	"github.com/h3poteto/fluentd-sidecar-injector/pkg/controller/sidecarinjector"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

var _ = Describe("E2E", func() {
	var (
		cfg *rest.Config
	)
	BeforeSuite(func() {
		// Deploy operator controller
		configfile := os.Getenv("KUBECONFIG")
		if configfile == "" {
			configfile = "$HOME/.kube/config"
		}
		restConfig, err := clientcmd.BuildConfigFromFlags("", os.ExpandEnv(configfile))
		if err != nil {
			panic(err)
		}
		cfg = restConfig

		client, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			panic(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := applyCRD(ctx, restConfig, client); err != nil {
			panic(err)
		}
		klog.Info("applying RBAC")
		if err := util.ApplyRBAC(ctx, restConfig); err != nil {
			panic(err)
		}
		klog.Info("applying manager")

		// Apply manager
		if err := applyManager(ctx, client, "default"); err != nil {
			panic(err)
		}

	})
	AfterSuite(func() {
		// Delete operator controller and custom resources
		configfile := os.Getenv("KUBECONFIG")
		if configfile == "" {
			configfile = "$HOME/.kube/config"
		}
		restConfig, err := clientcmd.BuildConfigFromFlags("", os.ExpandEnv(configfile))
		if err != nil {
			panic(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		client, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			panic(err)
		}
		if err := deleteManager(ctx, client, "default"); err != nil {
			panic(err)
		}

		if err := util.DeleteRBAC(ctx, restConfig); err != nil {
			panic(err)
		}
		if err := util.DeleteCRD(ctx, restConfig); err != nil {
			panic(err)
		}

	})
	Describe("Operator", func() {
		// Check deploying custom resources
		It("Should be deployed a custom resource and created a webhook configuration", func() {
			ctx := context.Background()
			ownClient, err := clientset.NewForConfig(cfg)
			Expect(err).To(BeNil())
			sidecarInjector := fixtures.NewSidecarInjector("default")
			_, err = ownClient.OperatorV1alpha1().SidecarInjectors("default").Create(ctx, sidecarInjector, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			client, err := kubernetes.NewForConfig(cfg)
			Expect(err).To(BeNil())

			var webhook *admissionregistrationv1.MutatingWebhookConfiguration
			err = wait.Poll(10*time.Second, 5*time.Minute, func() (bool, error) {
				res, err := client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, sidecarinjector.MutatingNamePrefix+sidecarInjector.Name, metav1.GetOptions{})
				if err != nil {
					if kerrors.IsNotFound(err) {
						return false, nil
					}
					return false, err
				}
				if res == nil {
					return false, nil
				}
				webhook = res
				return true, nil
			})
			Expect(err).To(BeNil())
			Expect(webhook).NotTo(BeNil())
		})
	})
	Describe("Webhook", func() {
		// Check webhook are injected
	})
})

func applyCRD(ctx context.Context, cfg *rest.Config, client *kubernetes.Clientset) error {
	klog.Info("applying CRD")
	err := util.ApplyCRD(ctx, cfg)
	if err != nil {
		panic(err)
	}
	time.Sleep(10 * time.Second)
	return err
}

func applyManager(ctx context.Context, client *kubernetes.Clientset, ns string) error {
	sa, clusterRoleBinding, role, roleBinding, deployment := fixtures.NewManagerManifests(ns, "sidecar-injector-manager-role", "ghcr.io/h3poteto/fluentd-sidecar-injector:latest")
	if _, err := client.CoreV1().ServiceAccounts(ns).Create(ctx, sa, metav1.CreateOptions{}); err != nil {
		return err
	}
	if _, err := client.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{}); err != nil {
		return err
	}
	if _, err := client.RbacV1().Roles(ns).Create(ctx, role, metav1.CreateOptions{}); err != nil {
		return err
	}
	if _, err := client.RbacV1().RoleBindings(ns).Create(ctx, roleBinding, metav1.CreateOptions{}); err != nil {
		return err
	}
	if _, err := client.AppsV1().Deployments(ns).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
		return err
	}

	err := wait.Poll(10*time.Second, 5*time.Minute, func() (bool, error) {
		podList, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", fixtures.ManagerPodLabelKey, fixtures.ManagerPodLabelValue),
		})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		klog.V(4).Infof("Pods are %#v", podList.Items)
		if len(podList.Items) == 0 {
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
	})
	if err != nil {
		return err
	}

	return nil
}

func deleteManager(ctx context.Context, client *kubernetes.Clientset, ns string) error {
	sa, clusterRoleBinding, role, roleBinding, deployment := fixtures.NewManagerManifests(ns, "sidecar-injector-manager-role", "ghcr.io/h3poteto/fluentd-sidecar-injector:latest")
	if err := client.AppsV1().Deployments(ns).Delete(ctx, deployment.Name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	if err := client.RbacV1().RoleBindings(ns).Delete(ctx, roleBinding.Name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	if err := client.RbacV1().Roles(ns).Delete(ctx, role.Name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	if err := client.RbacV1().ClusterRoleBindings().Delete(ctx, clusterRoleBinding.Name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	if err := client.CoreV1().ServiceAccounts(ns).Delete(ctx, sa.Name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	err := wait.Poll(10*time.Second, 10*time.Minute, func() (bool, error) {
		podList, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", fixtures.ManagerPodLabelKey, fixtures.ManagerPodLabelValue),
		})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		klog.V(4).Infof("Pods are: %#v", podList.Items)
		if len(podList.Items) == 0 {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return err
	}

	return nil
}
