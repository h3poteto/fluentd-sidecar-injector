package sidecarinjector

import (
	_ "embed"
)

//go:embed templates/issuer.yaml.tmpl
var issuerTmpl string

//go:embed templates/certificate.yaml.tmpl
var certificateTmpl string
