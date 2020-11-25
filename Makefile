.PHONY: certs codegen

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

CRD_OPTIONS ?= "crd:trivialVersions=true"

run: codegen manifests
	go run ./main.go controller sidecar-injector

install: manifests
	kubectl apply -f ./config/crd

uninstall: manifests
	kubectl delete -f ./config/crd

clean:
	rm ./install/kustomize/base/certs/*.key
	rm ./install/kustomize/base/certs/*.csr
	rm ./install/kustomize/base/certs/*.crt
	rm ./*.crt
	rm ./*.key

codegen:
	${GOPATH}/src/k8s.io/code-generator/generate-groups.sh "deepcopy,client,informer,lister" \
	github.com/h3poteto/fluentd-sidecar-injector/pkg/client github.com/h3poteto/fluentd-sidecar-injector/pkg/apis \
	sidecarinjectorcontroller:v1alpha1

manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=sidecar-injector-manager-role paths=./...  output:crd:artifacts:config=./config/crd/

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
