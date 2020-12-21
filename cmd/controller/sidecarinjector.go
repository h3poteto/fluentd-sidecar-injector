package controller

import (
	"context"
	"os"
	"time"

	clientset "github.com/h3poteto/fluentd-sidecar-injector/pkg/client/clientset/versioned"
	informers "github.com/h3poteto/fluentd-sidecar-injector/pkg/client/informers/externalversions"
	"github.com/h3poteto/fluentd-sidecar-injector/pkg/controller/sidecarinjector"
	"github.com/h3poteto/fluentd-sidecar-injector/pkg/leaderelection"
	"github.com/spf13/cobra"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

type sidecarInjectorOption struct {
}

func sidecarInjectorCmd() *cobra.Command {
	o := &sidecarInjectorOption{}
	cmd := &cobra.Command{
		Use:   "sidecar-injector",
		Short: "Start SidecarInjector controller",
		Run:   o.run,
	}

	return cmd
}

func (o *sidecarInjectorOption) run(cmd *cobra.Command, args []string) {
	kubeconfig, masterURL := controllerConfig()
	if kubeconfig != "" {
		klog.Infof("Using kubeconfig: %s", kubeconfig)
	}
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	ns := os.Getenv("POD_NAMESPACE")
	if ns == "" {
		ns = "default"
	}
	le := leaderelection.NewLeaderElection("sidecar-injector", ns)
	ctx := context.Background()
	err = le.Run(ctx, cfg, func(ctx context.Context, clientConfig *rest.Config, stopCh <-chan struct{}) {
		kubeClient, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
		}

		ownClient, err := clientset.NewForConfig(cfg)
		if err != nil {
			klog.Fatalf("Error building own clientset: %s", err.Error())
		}

		kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
		ownInformerFactory := informers.NewSharedInformerFactory(ownClient, time.Second*30)

		controller := sidecarinjector.NewController(kubeClient, ownClient,
			kubeInformerFactory.Apps().V1().Deployments(),
			kubeInformerFactory.Core().V1().Secrets(),
			kubeInformerFactory.Core().V1().Services(),
			kubeInformerFactory.Admissionregistration().V1().MutatingWebhookConfigurations(),
			ownInformerFactory.Operator().V1alpha1().SidecarInjectors())

		go kubeInformerFactory.Start(stopCh)
		go ownInformerFactory.Start(stopCh)

		if err = controller.Run(2, stopCh); err != nil {
			klog.Fatalf("Error running controller: %s", err.Error())
		}
	})
	klog.Fatalf("Error starting controller: %s", err.Error())
}
