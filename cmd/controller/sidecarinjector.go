package controller

import (
	"log"

	"github.com/spf13/cobra"
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
	kubeconfig := controllerConfig()
	log.Println(kubeconfig)
}
