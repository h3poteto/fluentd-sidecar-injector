package cmd

import "github.com/spf13/cobra"

type server struct {
	tlsCertFile string
	tlsKeyFile  string
}

func serverCmd() *cobra.Command {
	s := &server{}
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start webhook server",
		Run:   s.run,
	}
	flags := cmd.Flags()
	flags.StringVarP(&s.tlsCertFile, "tls-cert-file", "c", "", "Certificate file name of TLS")
	flags.StringVarP(&s.tlsKeyFile, "tls-key-file", "k", "", "Key file name of TLS")

	return cmd
}

func (s *server) run(cmd *cobra.Command, args []string) {

}
