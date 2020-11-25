package controller

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ControllerCmd command for controllers.
func ControllerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "controller",
		Short: "Start custom controller",
	}

	cmd.PersistentFlags().String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	cmd.PersistentFlags().String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	_ = viper.BindPFlag("kubeconfig", cmd.PersistentFlags().Lookup("kubeconfig"))
	_ = viper.BindPFlag("master", cmd.PersistentFlags().Lookup("master"))

	cmd.AddCommand(sidecarInjectorCmd())

	return cmd
}

func controllerConfig() (string, string) {
	kubeconfig := viper.GetString("kubeconfig")
	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = "$HOME/.kube/config"
			if _, err := os.Stat(kubeconfig); err != nil {
				kubeconfig = ""
			}
		}
	}
	master := viper.GetString("master")
	return kubeconfig, master
}
