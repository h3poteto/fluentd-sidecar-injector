package cmd

import (
	"github.com/h3poteto/fluentd-sidecar-injector/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type serverOption struct {
	tlsCertFile string
	tlsKeyFile  string
}

func serverCmd() *cobra.Command {
	s := &serverOption{}
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

func (o *serverOption) run(cmd *cobra.Command, args []string) {
	if o.tlsCertFile == "" {
		logrus.Fatal("tls-cert-file is required parameter")
	}
	if o.tlsKeyFile == "" {
		logrus.Fatal("tls-key-file is required parameter")
	}

	if err := server.StartServer(o.tlsCertFile, o.tlsKeyFile); err != nil {
		logrus.Fatal(err)
	}
}
