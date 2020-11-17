.PHONY: certs

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
