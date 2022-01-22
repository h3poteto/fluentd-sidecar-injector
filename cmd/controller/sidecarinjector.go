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
	useCertManager bool
	workers        int
}

func sidecarInjectorCmd() *cobra.Command {
	o := &sidecarInjectorOption{}
	cmd := &cobra.Command{
		Use:   "sidecar-injector",
		Short: "Start SidecarInjector controller",
		Run:   o.run,
	}
	flags := cmd.Flags()
	flags.BoolVar(&o.useCertManager, "use-cert-manager", false, "If you already use cert-manager, please enable this flag. If false, this controller generates its own certificate for webhook server. ")
	flags.IntVarP(&o.workers, "workers", "w", 1, "Concurrent workers number for controller.")

	return cmd
}

func (o *sidecarInjectorOption) run(cmd *cobra.Command, args []string) {
	kubeconfig, masterURL := controllerConfig()
	if kubeconfig != "" {
		klog.Infof("Using kubeconfig: %s", kubeconfig)
	} else {
		klog.Info("Using in-cluster config")
	}
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building rest config: %s", err.Error())
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

		dynamicClient, err := sidecarinjector.NewDynamicClient(clientConfig, kubeClient)
		if err != nil {
			klog.Fatalf("Failed to build dynamic client: %s", err.Error())
		}

		kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
		ownInformerFactory := informers.NewSharedInformerFactory(ownClient, time.Second*30)

		controller := sidecarinjector.NewController(
			kubeClient,
			ownClient,
			dynamicClient,
			kubeInformerFactory,
			ownInformerFactory,
			o.useCertManager,
		)

		go kubeInformerFactory.Start(stopCh)
		go ownInformerFactory.Start(stopCh)

		if err = controller.Run(o.workers, stopCh); err != nil {
			klog.Fatalf("Error running controller: %s", err.Error())
		}
	})
	klog.Fatalf("Error starting controller: %s", err.Error())
}
