#!/bin/bash

function generate {
    CA_BUNDLE=$(cat ./install/kustomize/base/certs/webhook.crt | base64 -w0)
    NAMESPACE=$1
    sed -e "s/CA_BUNDLE/${CA_BUNDLE}/" -e "s/NAMESPACE/${NAMESPACE}/" ./install/kustomize/mutating-webhook-configuration.yaml.tpl > ./install/kustomize/mutating-webhook-configuration.yaml
    sed -e "s/NAMESPACE/${NAMESPACE}/" ./install/kustomize/kustomization.yaml.tpl > ./install/kustomize/kustomization.yaml
}

if [ $# -ne 1 ]; then
    echo "generate_template <namespace>"
    exit 1
fi

generate $1
