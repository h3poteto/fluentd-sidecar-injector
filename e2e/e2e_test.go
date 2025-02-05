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
	pkgwebhook "github.com/h3poteto/fluentd-sidecar-injector/pkg/webhook/sidecarinjector"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

var (
	cfg       *rest.Config
	ownClient *clientset.Clientset
	client    *kubernetes.Clientset
	managerNS string
)

var _ = BeforeSuite(func() {
	managerNS = "kube-public"
	configfile := os.Getenv("KUBECONFIG")
	if configfile == "" {
		configfile = "$HOME/.kube/config"
	}
	var err error
	cfg, err = clientcmd.BuildConfigFromFlags("", os.ExpandEnv(configfile))
	Expect(err).ShouldNot(HaveOccurred())

	client, err = kubernetes.NewForConfig(cfg)
	Expect(err).ShouldNot(HaveOccurred())

	ownClient, err = clientset.NewForConfig(cfg)
	Expect(err).ShouldNot(HaveOccurred())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	err = waitUntilReady(ctx, client)
	Expect(err).ShouldNot(HaveOccurred())
})

var _ = Describe("E2E", func() {
	Describe("Webhook is created and sidecar containers are injected", func() {
		var (
			useCertManager bool
			collector      string
			webhook        *admissionregistrationv1.MutatingWebhookConfiguration
			setupError     error
		)

		JustBeforeEach(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			if err := applyCRD(ctx, cfg, client); err != nil {
				panic(err)
			}
			klog.Info("applying RBAC")
			if err := util.ApplyRBAC(ctx, cfg); err != nil {
				panic(err)
			}
			klog.Info("applying manager")

			// Apply manager
			if err := applyManager(ctx, client, managerNS, useCertManager); err != nil {
				panic(err)
			}

			webhook, setupError = applySidecarInjector(context.Background(), client, ownClient, collector)
		})

		AfterEach(func() {
			ctx := context.Background()
			err := deleteSidecarInjector(ctx, client, ownClient, collector)
			if err != nil {
				panic(err)
			}

			// Delete operator controller and custom resources
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			if err := deleteManager(ctx, client, managerNS); err != nil {
				panic(err)
			}

			if err := util.DeleteRBAC(ctx, cfg); err != nil {
				panic(err)
			}
			if err := util.DeleteCRD(ctx, cfg); err != nil {
				panic(err)
			}
		})
		Context("Use self managed certificate", func() {
			BeforeEach(func() {
				useCertManager = false
			})
			Context("Collector is fluentd", func() {
				BeforeEach(func() {
					collector = "fluentd"
				})
				It("fluentd container is injected", func() {
					spec(setupError, webhook, client, managerNS, "ghcr.io/h3poteto/fluentd-forward:latest")
				})
			})
			Context("Collector is fluent-bit", func() {
				BeforeEach(func() {
					collector = "fluent-bit"
				})
				It("fluent-bit container is injectd", func() {
					spec(setupError, webhook, client, managerNS, "ghcr.io/h3poteto/fluentbit-forward:latest")
				})
			})
		})

		Context("Use cert-manager", func() {
			BeforeEach(func() {
				useCertManager = true
			})
			Context("Collector is fluentd", func() {
				BeforeEach(func() {
					collector = "fluentd"
				})
				It("fluentd container is injected", func() {
					spec(setupError, webhook, client, managerNS, "ghcr.io/h3poteto/fluentd-forward:latest")
				})
			})
		})
	})
})

func waitUntilReady(ctx context.Context, client *kubernetes.Clientset) error {
	klog.Info("Waiting until kubernetes cluster is ready")
	err := wait.Poll(10*time.Second, 10*time.Minute, func() (bool, error) {
		nodeList, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to list nodes: %v", err)
		}
		if len(nodeList.Items) == 0 {
			klog.Warningf("node does not exist yet")
			return false, nil
		}
		for i := range nodeList.Items {
			n := &nodeList.Items[i]
			if !nodeIsReady(n) {
				klog.Warningf("node %s is not ready yet", n.Name)
				return false, nil
			}
		}
		klog.Info("all nodes are ready")
		return true, nil
	})
	return err
}

func nodeIsReady(node *corev1.Node) bool {
	for i := range node.Status.Conditions {
		con := &node.Status.Conditions[i]
		if con.Type == corev1.NodeReady && con.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func applyCRD(ctx context.Context, cfg *rest.Config, client *kubernetes.Clientset) error {
	klog.Info("applying CRD")
	err := util.ApplyCRD(ctx, cfg)
	if err != nil {
		panic(err)
	}
	time.Sleep(10 * time.Second)
	return err
}

func applyManager(ctx context.Context, client *kubernetes.Clientset, ns string, useCertManager bool) error {
	image := os.Getenv("FLUENTD_SIDECAR_INJECTOR_IMAGE")
	if image == "" {
		return fmt.Errorf("FLUENTD_SIDECAR_INJECTOR_IMAGE is required")
	}
	sa, clusterRoleBinding, role, roleBinding, deployment := fixtures.NewManagerManifests(ns, "sidecar-injector-manager-role", image, useCertManager)
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

		return util.WaitPodRunning(podList)
	})
	if err != nil {
		return err
	}

	return nil
}

func deleteManager(ctx context.Context, client *kubernetes.Clientset, ns string) error {
	image := os.Getenv("FLUENTD_SIDECAR_INJECTOR_IMAGE")
	if image == "" {
		return fmt.Errorf("FLUENTD_SIDECAR_INJECTOR_IMAGE is required")
	}
	sa, clusterRoleBinding, role, roleBinding, deployment := fixtures.NewManagerManifests(ns, "sidecar-injector-manager-role", image, false)
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

func applyTestPod(ctx context.Context, client *kubernetes.Clientset, ns string) (*appsv1.Deployment, error) {
	nginx := fixtures.NewNginx(ns)
	return client.AppsV1().Deployments(ns).Create(ctx, nginx, metav1.CreateOptions{})
}

func deleteTestPod(ctx context.Context, client *kubernetes.Clientset, ns string) error {
	nginx := fixtures.NewNginx(ns)
	return client.AppsV1().Deployments(ns).Delete(ctx, nginx.Name, metav1.DeleteOptions{})
}

func applySidecarInjector(ctx context.Context, client *kubernetes.Clientset, ownClient *clientset.Clientset, collector string) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
	sidecarInjector := fixtures.NewSidecarInjector(collector)
	_, err := ownClient.OperatorV1alpha1().SidecarInjectors().Create(ctx, sidecarInjector, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	var webhook *admissionregistrationv1.MutatingWebhookConfiguration
	err = wait.Poll(3*time.Second, 5*time.Minute, func() (bool, error) {
		res, err := client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, sidecarinjector.MutatingNamePrefix+sidecarInjector.Name, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			klog.Error(err)
			return false, err
		}
		if res == nil {
			return false, nil
		}
		webhook = res
		return true, nil
	})
	return webhook, err
}

func deleteSidecarInjector(ctx context.Context, client *kubernetes.Clientset, ownClient *clientset.Clientset, collector string) error {
	sidecarInjector := fixtures.NewSidecarInjector(collector)
	if err := ownClient.OperatorV1alpha1().SidecarInjectors().Delete(ctx, sidecarInjector.Name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	err := wait.Poll(3*time.Second, 5*time.Minute, func() (bool, error) {
		res, err := client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, sidecarinjector.MutatingNamePrefix+sidecarInjector.Name, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return true, nil
			}
			klog.Error(err)
			return false, err
		}
		if res == nil {
			return true, nil
		}
		klog.Warningf("webhook configuration %s is still living", res.Name)
		return false, nil
	})
	return err
}

func spec(
	setupError error,
	webhook *admissionregistrationv1.MutatingWebhookConfiguration,
	client *kubernetes.Clientset,
	ns,
	injectedContainerImage string) {
	Expect(setupError).To(BeNil())
	Expect(webhook).NotTo(BeNil())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Wait until webhook servers are deployed.
	err := wait.Poll(10*time.Second, 5*time.Minute, func() (bool, error) {
		podList, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", sidecarinjector.WebhookServerLabelKey, sidecarinjector.WebhookServerLabelValue),
		})
		if err != nil {
			if kerrors.IsNotFound(err) {
				klog.Info("Webhook servers have not been deployed yet")
				return false, nil
			}
			return false, err
		}
		return util.WaitPodRunning(podList)

	})
	Expect(err).To(BeNil())

	testPodNS := "default"
	_, err = applyTestPod(ctx, client, testPodNS)
	Expect(err).To(BeNil())

	var pods []corev1.Pod
	err = wait.Poll(10*time.Second, 5*time.Minute, func() (bool, error) {
		podList, err := client.CoreV1().Pods(testPodNS).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", fixtures.TestPodLabelKey, fixtures.TestPodLabelValue),
		})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		running, err := util.WaitPodRunning(podList)
		if running {
			pods = podList.Items
		}
		return running, err
	})
	Expect(err).To(BeNil())

	for i := range pods {
		Expect(len(pods[i].Spec.Containers)).To(Equal(2), "Containers count is not matched")
		// The default token secret is mounted, so the volume has been mounted before sidecar container is injected.
		Expect(len(pods[i].Spec.Volumes)).To(Equal(2), "Volumes count is not matched")
		volume := util.FindVolume(pods[i].Spec.Volumes, pkgwebhook.VolumeName)
		Expect(volume).NotTo(BeNil(), "Pod volume is not matched")

		container := util.FindContainer(&pods[i], pkgwebhook.ContainerName)
		Expect(container).NotTo(BeNil(), "Sidecar container is not found")
		Expect(container.Image).To(Equal(injectedContainerImage), "Injectd image is not matched")
		containerVolume := util.FindMount(container.VolumeMounts, pkgwebhook.VolumeName)
		Expect(containerVolume).NotTo(BeNil(), "Volume is not mounted to sidecar container")
		Expect(containerVolume.MountPath).To(Equal(fixtures.LogDir), "Sidecar container volume mount is not matched")

		nginx := util.FindContainer(&pods[i], fixtures.TestContainerName)
		Expect(nginx).NotTo(BeNil(), "Nginx container is not found")
		nginxVolume := util.FindMount(nginx.VolumeMounts, pkgwebhook.VolumeName)
		Expect(nginxVolume).NotTo(BeNil(), "Volume is not mounted to nginx container")
		Expect(nginxVolume.MountPath).To(Equal(fixtures.LogDir), "Nginx container volume mount is not matched")
	}

	err = deleteTestPod(ctx, client, testPodNS)
	Expect(err).To(BeNil())

	err = wait.Poll(10*time.Second, 5*time.Minute, func() (bool, error) {
		podList, err := client.CoreV1().Pods(testPodNS).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", fixtures.TestPodLabelKey, fixtures.TestPodLabelValue),
		})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		if len(podList.Items) == 0 {
			return true, nil
		}
		for i := range podList.Items {
			pod := &podList.Items[i]
			klog.Warningf("pod %s/%s is still living", pod.Namespace, pod.Name)
		}
		return false, nil
	})
	Expect(err).To(BeNil())
}
