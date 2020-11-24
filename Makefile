.PHONY: certs codegen

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

CRD_OPTIONS ?= "crd:trivialVersions=true"

### DEPRECATED
NAMESPACE = default
SERVICE = fluentd-sidecar-injector-webhook

prepare:
	sed -e "s/SERVICE/${SERVICE}.${NAMESPACE}.svc/" ./install/kustomize/base/certs/san.tpl > ./install/kustomize/base/certs/san.txt
	openssl genrsa -out ./install/kustomize/base/certs/webhookCA.key 2048
	openssl req -new -key ./install/kustomize/base/certs/webhookCA.key -subj "/CN=${SERVICE}.${NAMESPACE}.svc" -out ./install/kustomize/base/certs/webhookCA.csr
	openssl x509 -req -days 365 -in ./install/kustomize/base/certs/webhookCA.csr -signkey ./install/kustomize/base/certs/webhookCA.key -out ./install/kustomize/base/certs/webhook.crt -extfile ./install/kustomize/base/certs/san.txt

build: prepare
	./generate_template.sh ${NAMESPACE}
	kubectl kustomize ./install/kustomize > kustomize.yaml

clean:
	rm ./install/kustomize/base/certs/*.key
	rm ./install/kustomize/base/certs/*.csr
	rm ./install/kustomize/base/certs/*.crt
### DEPRECATED

codegen:
	${GOPATH}/src/k8s.io/code-generator/generate-groups.sh "deepcopy,client,informer,lister" \
	github.com/h3poteto/fluentd-sidecar-injector/pkg/client github.com/h3poteto/fluentd-sidecar-injector/pkg/apis \
	sidecarinjectorcontroller:v1alpha1

manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=sidecar-injector-manager-role paths=./... output:dir=./crd

controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
