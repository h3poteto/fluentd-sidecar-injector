package dev

import "github.com/spf13/cobra"

type devOption struct{}

func DevCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Useful commands for developers",
	}
	cmd.AddCommand(certificateCmd())

	return cmd
}
