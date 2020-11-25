package cmd

import (
	"github.com/h3poteto/fluentd-sidecar-injector/cmd/controller"
	"github.com/h3poteto/fluentd-sidecar-injector/cmd/dev"
	"github.com/spf13/cobra"
)

// RootCmd is cobra command.
var RootCmd = &cobra.Command{
	Use:           "fluentd-sidecar-injector",
	Short:         "fluentd-sidecar-injector is a webhook server to inject fluentd sidecar",
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	cobra.OnInitialize()
	RootCmd.AddCommand(
		webhookCmd(),
		versionCmd(),
		controller.ControllerCmd(),
		dev.DevCmd(),
	)
}
