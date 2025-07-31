package sidecarinjector

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	sidecarinjectorv1alpha1 "github.com/h3poteto/fluentd-sidecar-injector/pkg/apis/sidecarinjectorcontroller/v1alpha1"
	clientset "github.com/h3poteto/fluentd-sidecar-injector/pkg/client/clientset/versioned"
	ownscheme "github.com/h3poteto/fluentd-sidecar-injector/pkg/client/clientset/versioned/scheme"
	informers "github.com/h3poteto/fluentd-sidecar-injector/pkg/client/informers/externalversions"
	listers "github.com/h3poteto/fluentd-sidecar-injector/pkg/client/listers/sidecarinjectorcontroller/v1alpha1"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	admissionregistrationlisters "k8s.io/client-go/listers/admissionregistration/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

const controllerAgentName = "sidecar-injector-controller"
const secretNamePrefix = "sidecar-injector-certs-"
const serviceNamePrefix = "sidecar-injector-"
const MutatingNamePrefix = "sidecar-injector-webhook-"
const issuerNamePrefix = "sidecar-injector-issuer-"
const certificateNamePrefix = "sidecar-injecter-certificate-"

type Controller struct {
	kubeclientset kubernetes.Interface
	ownclientset  clientset.Interface
	dynamicClient *DynamicClient

	deploymentsLister     appslisters.DeploymentLister
	deploymentsSynced     cache.InformerSynced
	secretsLister         corelisters.SecretLister
	secretsSynced         cache.InformerSynced
	serviceLister         corelisters.ServiceLister
	serviceSynced         cache.InformerSynced
	mutatingLister        admissionregistrationlisters.MutatingWebhookConfigurationLister
	mutatingSynced        cache.InformerSynced
	sidecarInjectorLister listers.SidecarInjectorLister
	sidecarInjectorSynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface

	recorder record.EventRecorder

	useCertManager bool
}

// +kubebuilder:rbac:groups=operator.h3poteto.dev,resources=sidecarinjectors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.h3poteto.dev,resources=sidecarinjectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets;services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="admissionregistration.k8s.io",resources=mutatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="cert-manager.io",resources=issuers;certificates,verbs=get;list;watch;create;update;patch;delete

func NewController(
	kubeclientset kubernetes.Interface,
	ownclientset clientset.Interface,
	dynamicClient *DynamicClient,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	ownInformerFactory informers.SharedInformerFactory,
	useCertManager bool,
) *Controller {
	err := ownscheme.AddToScheme(scheme.Scheme)
	if err != nil {
		klog.Error(err)
		return nil
	}
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	deploymentInformer := kubeInformerFactory.Apps().V1().Deployments()
	secretInformer := kubeInformerFactory.Core().V1().Secrets()
	serviceInformer := kubeInformerFactory.Core().V1().Services()
	mutatingInformer := kubeInformerFactory.Admissionregistration().V1().MutatingWebhookConfigurations()
	sidecarInjectorInformer := ownInformerFactory.Operator().V1alpha1().SidecarInjectors()

	controller := &Controller{
		kubeclientset:         kubeclientset,
		ownclientset:          ownclientset,
		dynamicClient:         dynamicClient,
		deploymentsLister:     deploymentInformer.Lister(),
		deploymentsSynced:     deploymentInformer.Informer().HasSynced,
		secretsLister:         secretInformer.Lister(),
		secretsSynced:         secretInformer.Informer().HasSynced,
		serviceLister:         serviceInformer.Lister(),
		serviceSynced:         serviceInformer.Informer().HasSynced,
		mutatingLister:        mutatingInformer.Lister(),
		mutatingSynced:        mutatingInformer.Informer().HasSynced,
		sidecarInjectorLister: sidecarInjectorInformer.Lister(),
		sidecarInjectorSynced: sidecarInjectorInformer.Informer().HasSynced,
		workqueue:             workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), controllerAgentName),
		recorder:              recorder,
		useCertManager:        useCertManager,
	}

	klog.Infof("Setting up event handlers")
	// Set up an event handler for when SidecarInjector resources change
	sidecarInjectorInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueSidecarInjector,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueSidecarInjector(new)
		},
	})

	// Set up an event handler for when Deployment resources change. This
	// handler will lookup the owner of the given Deployment, and if it is
	// owned by a Foo resource will enqueue that Foo resource for
	// processing. This way, we don't need to implement custom logic for
	// handling Deployment resources.
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*appsv1.Deployment)
			oldDepl := old.(*appsv1.Deployment)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	secretInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newSec := new.(*corev1.Secret)
			oldSec := old.(*corev1.Secret)
			if newSec.ResourceVersion == oldSec.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	return controller
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Info("Starting SidecarInjector controller")

	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.sidecarInjectorSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) syncHandler(key string) error {
	ctx := context.Background()
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	sidecarInjector, err := c.sidecarInjectorLister.Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("sidecarInjector '%s' in workqueue no longer exists", key))
			return nil
		}

		return err
	}

	ownerNamespace := os.Getenv("POD_NAMESPACE")
	if ownerNamespace == "" {
		return fmt.Errorf("POD_NAMESPACE is required, so please set downward API")
	}

	secretName := secretNamePrefix + sidecarInjector.Name
	serviceName := serviceNamePrefix + sidecarInjector.Name
	mutatingName := MutatingNamePrefix + sidecarInjector.Name

	if c.useCertManager {
		// Iusser
		issuerName := issuerNamePrefix + sidecarInjector.Name
		certificateName := certificateNamePrefix + sidecarInjector.Name
		if err := c.applyIssuer(ctx, issuerName, ownerNamespace, sidecarInjector); err != nil {
			return err
		}
		// Certificate
		if err := c.applyCertificate(ctx, secretName, certificateName, serviceName, issuerName, ownerNamespace, sidecarInjector); err != nil {
			return err
		}
		// WebhookConfiguration
		mutating, err := c.mutatingLister.Get(mutatingName)
		if errors.IsNotFound(err) {
			mutating, err = c.createMutatingWebhookConfigurationWithCertManager(ctx, sidecarInjector, mutatingName, ownerNamespace, serviceName, certificateName)
		}
		if err != nil {
			klog.Error(err)
			return err
		}
		if !metav1.IsControlledBy(mutating, sidecarInjector) {
			msg := fmt.Sprintf("Resource %q already exists and is not managed by SidecarInjector", mutating.Name)
			c.recorder.Event(sidecarInjector, corev1.EventTypeWarning, "ErrResourceExists", msg)
			return fmt.Errorf("%s", msg)
		}
	} else {
		// Secrets and Certificate
		var serverCertificate []byte
		secret, err := c.secretsLister.Secrets(ownerNamespace).Get(secretName)
		if errors.IsNotFound(err) {
			secret, serverCertificate, err = c.createSecret(ctx, sidecarInjector, ownerNamespace, serviceName, secretName)
		}
		if err != nil {
			return err
		}
		if !metav1.IsControlledBy(secret, sidecarInjector) {
			msg := fmt.Sprintf("Resource %q already exists and is not managed by SidecarInjector", secret.Name)
			c.recorder.Event(sidecarInjector, corev1.EventTypeWarning, "ErrResourceExists", msg)
			return fmt.Errorf("%s", msg)
		}

		// WebhookConfiguration
		mutating, err := c.mutatingLister.Get(mutatingName)
		if errors.IsNotFound(err) {
			mutating, err = c.createMutatingWebhookConfiguration(ctx, sidecarInjector, mutatingName, ownerNamespace, serviceName, serverCertificate)
		}
		if err != nil {
			return err
		}
		if !metav1.IsControlledBy(mutating, sidecarInjector) {
			msg := fmt.Sprintf("Resource %q already exists and is not managed by SidecarInjector", mutating.Name)
			c.recorder.Event(sidecarInjector, corev1.EventTypeWarning, "ErrResourceExists", msg)
			return fmt.Errorf("%s", msg)
		}
	}

	// Deployment
	containerImage := os.Getenv("WEBHOOK_CONTAINER_IMAGE")
	if containerImage == "" {
		return fmt.Errorf("The environment variable WEBHOOK_CONTAINER_IMAGE is required, please set it")
	}
	var deployment *appsv1.Deployment
	deploymentName := sidecarInjector.Status.InjectorDeploymentName
	if deploymentName == "" {
		deployment, err = c.createDeployment(ctx, sidecarInjector, ownerNamespace, secretName, containerImage)
	} else {
		deployment, err = c.deploymentsLister.Deployments(ownerNamespace).Get(deploymentName)
		if errors.IsNotFound(err) {
			deployment, err = c.createDeployment(ctx, sidecarInjector, ownerNamespace, secretName, containerImage)
		}
	}
	if err != nil {
		klog.Error(err)
		return err
	}
	if !metav1.IsControlledBy(deployment, sidecarInjector) {
		msg := fmt.Sprintf("Resource %q already exists and is not managed by SidecarInjector", deployment.Name)
		c.recorder.Event(sidecarInjector, corev1.EventTypeWarning, "ErrResourceExists", msg)
		return fmt.Errorf("%s", msg)
	}

	// Service
	service, err := c.serviceLister.Services(ownerNamespace).Get(serviceName)
	if errors.IsNotFound(err) {
		service, err = c.createService(ctx, sidecarInjector, ownerNamespace, serviceName)
	}
	if err != nil {
		klog.Error(err)
		return err
	}
	if !metav1.IsControlledBy(service, sidecarInjector) {
		msg := fmt.Sprintf("Resource %q already exists and is not managed by SidecarInjector", service.Name)
		c.recorder.Event(sidecarInjector, corev1.EventTypeWarning, "ErrResourceExists", msg)
		return fmt.Errorf("%s", msg)
	}

	err = c.updateSidecarInjectorStatus(ctx, sidecarInjector, deployment, service)
	if err != nil {
		klog.Error(err)
		return err
	}

	c.recorder.Event(sidecarInjector, corev1.EventTypeNormal, "Synced", "SidecarInjector synced successfully")
	return nil
}

func (c *Controller) updateSidecarInjectorStatus(ctx context.Context, sidecarInjector *sidecarinjectorv1alpha1.SidecarInjector, deployment *appsv1.Deployment, service *corev1.Service) error {
	serviceReady := false
	if service != nil && len(service.Spec.Ports) > 0 && service.Spec.ClusterIP != "" {
		serviceReady = true
	}
	sidecarInjectorCopy := sidecarInjector.DeepCopy()
	sidecarInjectorCopy.Status.InjectorDeploymentName = deployment.Name
	sidecarInjectorCopy.Status.InjectorPodCount = deployment.Status.AvailableReplicas
	sidecarInjectorCopy.Status.InjectorServiceReady = serviceReady
	_, err := c.ownclientset.OperatorV1alpha1().SidecarInjectors().Update(ctx, sidecarInjectorCopy, metav1.UpdateOptions{})
	return err
}

func (c *Controller) enqueueSidecarInjector(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		if ownerRef.Kind != "SidecarInjector" {
			return
		}

		sidecarInjector, err := c.sidecarInjectorLister.Get(ownerRef.Name)
		if err != nil {
			klog.V(4).Infof("ignoring orphaned object '%s' of foo '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueueSidecarInjector(sidecarInjector)
		return
	}
}

func (c *Controller) createDeployment(ctx context.Context, sidecarInjector *sidecarinjectorv1alpha1.SidecarInjector, namespace, secretName, image string) (*appsv1.Deployment, error) {
	deployment := newDeployment(sidecarInjector, namespace, secretName, image)
	return c.kubeclientset.AppsV1().Deployments(deployment.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
}

func (c *Controller) createSecret(ctx context.Context, sidecarInjector *sidecarinjectorv1alpha1.SidecarInjector, namespace, serviceName, secretName string) (*corev1.Secret, []byte, error) {
	secret, serverCertificate, err := newSecret(sidecarInjector, namespace, serviceName, secretName)
	if err != nil {
		return nil, nil, err
	}
	res, err := c.kubeclientset.CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return nil, nil, err
	}
	return res, serverCertificate, nil
}

func (c *Controller) createService(ctx context.Context, sidecarInjector *sidecarinjectorv1alpha1.SidecarInjector, namespace, serviceName string) (*corev1.Service, error) {
	service := newService(sidecarInjector, namespace, serviceName)
	return c.kubeclientset.CoreV1().Services(service.Namespace).Create(ctx, service, metav1.CreateOptions{})
}

func (c *Controller) createMutatingWebhookConfiguration(ctx context.Context, sidecarInjector *sidecarinjectorv1alpha1.SidecarInjector, mutatingName, namespace, serviceName string, serverCetriicate []byte) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
	mutating := newMutatingWebhookConfiguration(sidecarInjector, mutatingName, namespace, serviceName)
	for i := range mutating.Webhooks {
		mutating.Webhooks[i].ClientConfig.CABundle = serverCetriicate
	}
	return c.kubeclientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(ctx, mutating, metav1.CreateOptions{})
}

func (c *Controller) createMutatingWebhookConfigurationWithCertManager(ctx context.Context, sidecarInjector *sidecarinjectorv1alpha1.SidecarInjector, mutatingName, namespace, serviceName, certificateName string) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
	mutating := newMutatingWebhookConfiguration(sidecarInjector, mutatingName, namespace, serviceName)
	mutating.Annotations["cert-manager.io/inject-ca-from"] = namespace + "/" + certificateName

	return c.kubeclientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(ctx, mutating, metav1.CreateOptions{})
}

func (c *Controller) applyIssuer(ctx context.Context, issuerName, namespace string, sidecarInjector *sidecarinjectorv1alpha1.SidecarInjector) error {
	ownerRef := metav1.NewControllerRef(sidecarInjector, schema.GroupVersionKind{
		Group:   sidecarinjectorv1alpha1.SchemeGroupVersion.Group,
		Version: sidecarinjectorv1alpha1.SchemeGroupVersion.Version,
		Kind:    "SidecarInjector",
	})
	manifest, err := issuerManifest(issuerName, namespace, ownerRef)
	if err != nil {
		klog.Error(err)
		return err
	}

	return c.applyManifest(ctx, manifest)
}

func (c *Controller) applyCertificate(ctx context.Context, secretName, certificateName, serviceName, issuerName, namespace string, sidecarInjector *sidecarinjectorv1alpha1.SidecarInjector) error {
	ownerRef := metav1.NewControllerRef(sidecarInjector, schema.GroupVersionKind{
		Group:   sidecarinjectorv1alpha1.SchemeGroupVersion.Group,
		Version: sidecarinjectorv1alpha1.SchemeGroupVersion.Version,
		Kind:    "SidecarInjector",
	})
	manifest, err := certificateManifest(secretName, certificateName, serviceName, issuerName, namespace, ownerRef)
	if err != nil {
		klog.Error(err)
		return err
	}

	return c.applyManifest(ctx, manifest)
}

func (c *Controller) applyManifest(ctx context.Context, manifest *bytes.Buffer) error {
	decoder := yaml.NewYAMLOrJSONDecoder(manifest, 100)
	for {
		var rawObj runtime.RawExtension
		if err := decoder.Decode(&rawObj); err != nil {
			break
		}

		obj := &unstructured.Unstructured{}
		client, err := c.dynamicClient.ResourceClient(rawObj.Raw, obj)
		if err != nil {
			klog.Error(err)
			return err
		}
		_, err = c.dynamicClient.Get(ctx, client, obj)
		if errors.IsNotFound(err) {
			if _, err = c.dynamicClient.Apply(ctx, client, obj); err != nil {
				klog.Error(err)
				return err
			}
		} else if err != nil {
			klog.Error(err)
			return err
		}
	}
	return nil
}
