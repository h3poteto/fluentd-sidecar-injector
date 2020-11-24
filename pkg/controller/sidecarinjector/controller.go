package sidecarinjector

import (
	"context"
	"fmt"
	"time"

	sidecarinjectorv1alpha1 "github.com/h3poteto/fluentd-sidecar-injector/pkg/apis/sidecarinjectorcontroller/v1alpha1"
	clientset "github.com/h3poteto/fluentd-sidecar-injector/pkg/client/clientset/versioned"
	ownscheme "github.com/h3poteto/fluentd-sidecar-injector/pkg/client/clientset/versioned/scheme"
	informers "github.com/h3poteto/fluentd-sidecar-injector/pkg/client/informers/externalversions/sidecarinjectorcontroller/v1alpha1"
	listers "github.com/h3poteto/fluentd-sidecar-injector/pkg/client/listers/sidecarinjectorcontroller/v1alpha1"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	admissionregistrationinformers "k8s.io/client-go/informers/admissionregistration/v1"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
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
const dockerImage = "ghcr.io/h3poteto/fluentd-sidecar-injector"
const secretNamePrefix = "sidecar-injector-certs-"
const serviceNamePrefix = "sidecar-injector-"
const mutatingNamePrefix = "sidecar-injector-webhook-"

type Controller struct {
	kubeclientset kubernetes.Interface
	ownclientset  clientset.Interface

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
}

// +kubebuilder:rbac:groups=operator.h3poteto.dev,resources=sidecarinjectors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.h3poteto.dev,resources=sidecarinjectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="admissionregistration.k8s.io",resources=mutatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func NewController(
	kubeclientset kubernetes.Interface,
	ownclientset clientset.Interface,
	deploymentInformer appsinformers.DeploymentInformer,
	secretInformer coreinformers.SecretInformer,
	serviceInformer coreinformers.ServiceInformer,
	mutatingInformer admissionregistrationinformers.MutatingWebhookConfigurationInformer,
	sidecarInjectorInformer informers.SidecarInjectorInformer) *Controller {

	ownscheme.AddToScheme(scheme.Scheme)
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:         kubeclientset,
		ownclientset:          ownclientset,
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
	defer runtime.HandleCrash()
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
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
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
		runtime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) syncHandler(key string) error {
	ctx := context.Background()
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	sidecarInjector, err := c.sidecarInjectorLister.SidecarInjectors(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("sidecarInjector '%s' in workqueue no longer exists", key))
			return nil
		}

		return err
	}

	secretName := secretNamePrefix + sidecarInjector.Name
	serviceName := serviceNamePrefix + sidecarInjector.Name
	mutatingName := mutatingNamePrefix + sidecarInjector.Name

	var serverCertificate []byte
	secret, err := c.secretsLister.Secrets(sidecarInjector.Namespace).Get(secretName)
	if err != nil {
		if errors.IsNotFound(err) {
			secret, serverCertificate, err = c.createSecret(ctx, sidecarInjector, serviceName, secretName)
		}
	}
	if err != nil {
		return err
	}

	if !metav1.IsControlledBy(secret, sidecarInjector) {
		msg := fmt.Sprintf("Resource %q already exists and is not managed by SidecarInjector", secret.Name)
		c.recorder.Event(sidecarInjector, corev1.EventTypeWarning, "ErrResourceExists", msg)
		return fmt.Errorf(msg)
	}

	var deployment *appsv1.Deployment
	deploymentName := sidecarInjector.Status.InjectorDeploymentName
	if deploymentName == "" {
		deployment, err = c.createDeployment(ctx, sidecarInjector, secret.Name)
	} else {
		deployment, err = c.deploymentsLister.Deployments(sidecarInjector.Namespace).Get(deploymentName)
		if err != nil {
			if errors.IsNotFound(err) {
				deployment, err = c.createDeployment(ctx, sidecarInjector, secret.Name)
			}
		}
	}

	if err != nil {
		return err
	}

	if !metav1.IsControlledBy(deployment, sidecarInjector) {
		msg := fmt.Sprintf("Resource %q already exists and is not managed by SidecarInjector", deployment.Name)
		c.recorder.Event(sidecarInjector, corev1.EventTypeWarning, "ErrResourceExists", msg)
		return fmt.Errorf(msg)
	}

	service, err := c.serviceLister.Services(sidecarInjector.Namespace).Get(serviceName)
	if err != nil {
		if errors.IsNotFound(err) {
			service, err = c.createService(ctx, sidecarInjector, serviceName)
		}
	}

	if err != nil {
		return err
	}

	if !metav1.IsControlledBy(service, sidecarInjector) {
		msg := fmt.Sprintf("Resource %q already exists and is not managed by SidecarInjector", service.Name)
		c.recorder.Event(sidecarInjector, corev1.EventTypeWarning, "ErrResourceExists", msg)
		return fmt.Errorf(msg)
	}

	mutating, err := c.mutatingLister.Get(mutatingName)
	if err != nil {
		if errors.IsNotFound(err) {
			mutating, err = c.createMutatingWebhookConfiguration(ctx, sidecarInjector, mutatingName, service.Name, serverCertificate)
		}
	}

	if err != nil {
		return err
	}

	if !metav1.IsControlledBy(mutating, sidecarInjector) {
		msg := fmt.Sprintf("Resource %q already exists and is not managed by SidecarInjector", mutating.Name)
		c.recorder.Event(sidecarInjector, corev1.EventTypeWarning, "ErrResourceExists", msg)
		return fmt.Errorf(msg)
	}

	err = c.updateSidecarInjectorStatus(ctx, sidecarInjector, deployment, service)
	if err != nil {
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
	_, err := c.ownclientset.OperatorV1alpha1().SidecarInjectors(sidecarInjector.Namespace).Update(ctx, sidecarInjectorCopy, metav1.UpdateOptions{})
	return err
}

func (c *Controller) enqueueSidecarInjector(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
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
			runtime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		if ownerRef.Kind != "SidecarInjector" {
			return
		}

		sidecarInjector, err := c.sidecarInjectorLister.SidecarInjectors(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			klog.V(4).Infof("ignoring orphaned object '%s' of foo '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueueSidecarInjector(sidecarInjector)
		return
	}
}

func (c *Controller) createDeployment(ctx context.Context, sidecarInjector *sidecarinjectorv1alpha1.SidecarInjector, secretName string) (*appsv1.Deployment, error) {
	deployment := newDeployment(sidecarInjector, secretName)
	return c.kubeclientset.AppsV1().Deployments(sidecarInjector.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
}

func (c *Controller) createSecret(ctx context.Context, sidecarInjector *sidecarinjectorv1alpha1.SidecarInjector, serviceName, secretName string) (*corev1.Secret, []byte, error) {
	secret, serverCertificate, err := newSecret(sidecarInjector, serviceName, secretName)
	if err != nil {
		return nil, nil, err
	}
	res, err := c.kubeclientset.CoreV1().Secrets(sidecarInjector.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return nil, nil, err
	}
	return res, serverCertificate, nil
}

func (c *Controller) createService(ctx context.Context, sidecarInjector *sidecarinjectorv1alpha1.SidecarInjector, serviceName string) (*corev1.Service, error) {
	service := newService(sidecarInjector, serviceName)
	return c.kubeclientset.CoreV1().Services(sidecarInjector.Namespace).Create(ctx, service, metav1.CreateOptions{})
}

func (c *Controller) createMutatingWebhookConfiguration(ctx context.Context, sidecarInjector *sidecarinjectorv1alpha1.SidecarInjector, mutatingName, serviceName string, serverCetriicate []byte) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
	mutating := newMutatingWebhookConfiguration(sidecarInjector, mutatingName, serviceName, serverCetriicate)
	return c.kubeclientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(ctx, mutating, metav1.CreateOptions{})
}
