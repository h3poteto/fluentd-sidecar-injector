package cmd

import (
	"github.com/h3poteto/fluentd-sidecar-injector/pkg/webhook"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type webhookOption struct {
	tlsCertFile string
	tlsKeyFile  string
}

func webhookCmd() *cobra.Command {
	s := &webhookOption{}
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Start webhook server",
		Run:   s.run,
	}
	flags := cmd.Flags()
	flags.StringVarP(&s.tlsCertFile, "tls-cert-file", "c", "", "Certificate file name of TLS")
	flags.StringVarP(&s.tlsKeyFile, "tls-key-file", "k", "", "Key file name of TLS")

	return cmd
}

func (o *webhookOption) run(cmd *cobra.Command, args []string) {
	if o.tlsCertFile == "" {
		logrus.Fatal("tls-cert-file is required parameter")
	}
	if o.tlsKeyFile == "" {
		logrus.Fatal("tls-key-file is required parameter")
	}

	if err := webhook.Server(int32(8080), o.tlsCertFile, o.tlsKeyFile); err != nil {
		logrus.Fatal(err)
	}
}
