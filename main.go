package main

import (
	"fmt"
	"os"

	"github.com/h3poteto/fluentd-sidecar-injector/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
