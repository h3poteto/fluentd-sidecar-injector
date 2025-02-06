package webhook

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/h3poteto/fluentd-sidecar-injector/pkg/webhook/sidecarinjector"
	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	klog "k8s.io/klog/v2"
)

func Server(port int32, tlsCertFile, tlsKeyFile string) error {
	http.HandleFunc("/healthz", Healthz)
	http.HandleFunc("/mutate", ValidateSidecarInjector)

	listen := fmt.Sprintf(":%d", port)
	ssl := tlsCertFile != "" && tlsKeyFile != ""

	klog.Infof("Listening on %s, SSL is %t", listen, ssl)

	var err error
	if !ssl {
		err = http.ListenAndServe(listen, nil)
	} else {
		err = http.ListenAndServeTLS(listen, tlsCertFile, tlsKeyFile, nil)
	}
	return err
}

func Healthz(w http.ResponseWriter, r *http.Request) {
	klog.Infof("healthz")
	w.WriteHeader(http.StatusOK)
}

func ValidateSidecarInjector(w http.ResponseWriter, r *http.Request) {
	klog.Infof("validate-sidecarinjector")
	in, err := parseRequest(*r)
	if err != nil {
		klog.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := sidecarinjector.Validate(in)
	out, err := response.ToJSON()
	if err != nil {
		klog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(out)
	if err != nil {
		klog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func parseRequest(r http.Request) (*sidecarinjector.AdmissionReviewRequest, error) {
	if r.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("invalid Content-Type")
	}

	bodybuf := new(bytes.Buffer)
	_, err := bodybuf.ReadFrom(r.Body)
	if err != nil {
		return nil, err
	}
	body := bodybuf.Bytes()

	if len(body) == 0 {
		return nil, fmt.Errorf("empty body")
	}

	review, err := parseAdmissionReview(body)
	if err != nil {
		return nil, err

	}

	if review.Request == nil {
		return nil, fmt.Errorf("empty request")
	}

	return review, nil
}

func parseAdmissionReview(body []byte) (*sidecarinjector.AdmissionReviewRequest, error) {
	r := runtime.NewScheme()
	r.AddKnownTypes(admissionv1beta1.SchemeGroupVersion, &admissionv1beta1.AdmissionReview{})
	r.AddKnownTypes(admissionv1.SchemeGroupVersion, &admissionv1.AdmissionReview{})

	codecs := serializer.NewCodecFactory(r)
	review, _, err := codecs.UniversalDeserializer().Decode(body, nil, nil)
	if err != nil {
		return nil, err
	}

	switch ar := review.(type) {
	case *admissionv1beta1.AdmissionReview:
		return &sidecarinjector.AdmissionReviewRequest{
			TypeMeta: ar.TypeMeta,
			Request: &sidecarinjector.AdmissionRequest{
				UID:       ar.Request.UID,
				Kind:      ar.Request.Kind,
				Reousrce:  ar.Request.Resource,
				Name:      ar.Request.Name,
				Namespace: ar.Request.Namespace,
				Operation: sidecarinjector.AdmissionOperation(ar.Request.Operation),
				Object:    ar.Request.Object,
			},
		}, nil
	case *admissionv1.AdmissionReview:
		return &sidecarinjector.AdmissionReviewRequest{
			TypeMeta: ar.TypeMeta,
			Request: &sidecarinjector.AdmissionRequest{
				UID:       ar.Request.UID,
				Kind:      ar.Request.Kind,
				Reousrce:  ar.Request.Resource,
				Name:      ar.Request.Name,
				Namespace: ar.Request.Namespace,
				Operation: sidecarinjector.AdmissionOperation(ar.Request.Operation),
				Object:    ar.Request.Object,
			},
		}, nil
	default:
		return nil, fmt.Errorf("invalid admission review type")
	}
}
