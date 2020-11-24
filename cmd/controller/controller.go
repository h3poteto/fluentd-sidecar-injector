package controller

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type controllerOption struct {
	masterURL  string
	kubeconfig string
}

func ControllerCmd() *cobra.Command {
	o := &controllerOption{}
	cmd := &cobra.Command{
		Use:   "controller",
		Short: "Start custom controller",
	}

	flags := cmd.Flags()
	flags.StringVar(&o.kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flags.StringVar(&o.masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	viper.BindPFlag("kubeconfig", flags.Lookup("kubeconfig"))
	viper.BindPFlag("master", flags.Lookup("master"))

	cmd.AddCommand(sidecarInjectorCmd())

	return cmd
}

func controllerConfig() (string, string) {
	kubeconfig := viper.GetString("kubeconfig")
	if len(kubeconfig) == 0 {
		kubeconfig = os.Getenv("KUBECONFIG")
		if len(kubeconfig) == 0 {
			kubeconfig = "$HOME/.kube/config"
		}
	}
	master := viper.GetString("master")
	return kubeconfig, master
}
