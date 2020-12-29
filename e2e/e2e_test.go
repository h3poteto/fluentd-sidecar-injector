package e2e_test

import (
	"context"
	"os"
	"time"

	"github.com/h3poteto/fluentd-sidecar-injector/e2e/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := util.ApplyCRD(ctx, restConfig); err != nil {
			panic(err)
		}
		if err := util.ApplyRBAC(ctx, restConfig); err != nil {
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
