package dev

import "github.com/spf13/cobra"

// DevCmd commands for developer.
func DevCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Useful commands for developers",
	}
	cmd.AddCommand(certificateCmd())

	return cmd
}
