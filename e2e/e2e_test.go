package e2e_test

import (
	"context"
	"os"
	"time"

	"github.com/h3poteto/fluentd-sidecar-injector/e2e/pkg/fixtures"
	"github.com/h3poteto/fluentd-sidecar-injector/e2e/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var _ = Describe("E2E", func() {
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

		// Apply CRD
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := util.ApplyCRD(ctx, restConfig); err != nil {
			panic(err)
		}
		if err := util.ApplyRBAC(ctx, restConfig); err != nil {
			panic(err)
		}

		// Apply manager
		client, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			panic(err)
		}
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
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		client, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			panic(err)
		}
		if err := deleteManager(ctx, client, "default"); err != nil {
			panic(err)
		}

		if err := util.DeleteCRD(ctx, restConfig); err != nil {
			panic(err)
		}
		if err := util.DeleteRBAC(ctx, restConfig); err != nil {
			panic(err)
		}
	})
	Describe("Operator", func() {
		// Check deploying custom resources
		It("Sample", func() {
			Expect(true).To(Equal(true))
		})
	})
	Describe("Webhook", func() {
		// Check webhook are injected
	})
})

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

	return nil
}
