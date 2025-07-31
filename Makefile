.PHONY: certs codegen

# Get the currently used golang install path
# Use ~/.local/bin for local development if it exists in PATH, otherwise use go bin
ifneq (,$(findstring $(HOME)/.local/bin,$(PATH)))
GOBIN=$(HOME)/.local/bin
else
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif
endif

CRD_OPTIONS ?= "crd"
CODE_GENERATOR=${GOPATH}/src/k8s.io/code-generator
CODE_GENERATOR_TAG=v0.33.3
CONTROLLER_TOOLS_TAG=v0.18.0
BRANCH := $(shell git branch --show-current)

build: codegen manifests
	go build -a -tags netgo -installsuffix netgo -ldflags \
" \
  -extldflags '-static' \
  -X github.com/h3poteto/fluentd-sidecar-injector/cmd.version=$(shell git describe --tag --abbrev=0) \
  -X github.com/h3poteto/fluentd-sidecar-injector/cmd.revision=$(shell git rev-list -1 HEAD) \
  -X github.com/h3poteto/fluentd-sidecar-injector/cmd.build=$(shell git describe --tags) \
"

run: codegen manifests
	go run ./main.go controller sidecar-injector --kubeconfig=${KUBECONFIG}

install: manifests
	kubectl apply -f ./config/crd

uninstall: manifests
	kubectl delete -f ./config/crd

clean:
	rm -f ./*.crt
	rm -f ./*.key
	rm -f $(GOBIN)/controller-gen
	rm -rf $(CODE_GENERATOR)

# boilerplate is necessary to avoid: https://github.com/kubernetes/code-generator/issues/131
codegen: code-generator
	CODE_GENERATOR=${CODE_GENERATOR} scripts/update-codegen.sh

manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=sidecar-injector-manager-role paths=./...  output:crd:artifacts:config=./config/crd/

code-generator:
ifeq (, $(wildcard ${CODE_GENERATOR}))
	git clone https://github.com/kubernetes/code-generator.git ${CODE_GENERATOR} -b ${CODE_GENERATOR_TAG} --depth 1
endif

controller-gen:
ifeq (, $(shell which controller-gen))
	@echo "controller-gen not found, downloading..."
	curl -L -o controller-gen https://github.com/kubernetes-sigs/controller-tools/releases/download/${CONTROLLER_TOOLS_TAG}/controller-gen-linux-amd64
	chmod +x controller-gen
	mv controller-gen $(GOBIN)/controller-gen
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

gpush:
	docker build -f Dockerfile -t ghcr.io/h3poteto/fluentd-sidecar-injector:$(BRANCH) .
	docker push ghcr.io/h3poteto/fluentd-sidecar-injector:$(BRANCH)
