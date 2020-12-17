package dev

import (
	"encoding/pem"
	"os"

	"github.com/h3poteto/fluentd-sidecar-injector/pkg/controller/sidecarinjector"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

type certificateOption struct {
	outputKeyFile  string
	outputCertFile string
}

func certificateCmd() *cobra.Command {
	o := &certificateOption{}
	cmd := &cobra.Command{
		Use:   "certificate",
		Short: "Generate server certificates and save it to files",
		Run:   o.run,
	}

	flags := cmd.Flags()
	flags.StringVar(&o.outputKeyFile, "key-file", "server.key", "Path to a server key file name which you want to output.")
	flags.StringVar(&o.outputCertFile, "cert-file", "server.crt", "Path to a server certificate file name which you want to output.")

	return cmd
}

func (o *certificateOption) run(cmd *cobra.Command, args []string) {
	if o.outputKeyFile == "" {
		klog.Fatal("key-file argument is required")
	}
	if o.outputCertFile == "" {
		klog.Fatal("cert-file argument is required")
	}

	key, cert, err := sidecarinjector.NewCertificates("test-svc", "test-ns")
	if err != nil {
		klog.Fatal(err)
	}

	keyOut, err := os.Create(o.outputKeyFile)
	if err != nil {
		klog.Fatal(err)
	}
	defer keyOut.Close()
	keyPem, _ := pem.Decode(key)
	if err = pem.Encode(keyOut, keyPem); err != nil {
		klog.Fatal(err)
	}

	certOut, err := os.Create(o.outputCertFile)
	if err != nil {
		klog.Fatal(err)
	}
	defer certOut.Close()
	certPem, _ := pem.Decode(cert)
	if err = pem.Encode(certOut, certPem); err != nil {
		klog.Fatal(err)
	}
}
