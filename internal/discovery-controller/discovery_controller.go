package discoverycontroller

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	odinrest "odin-service-discovery-controller/internal/rest"
)

const (
	odinDiscoveryAddressAnnotationKey = "discovery.odin/address"
)

// CreateResource and DeleteResource labels
const (
	CreateResource = iota
	DeleteResource = iota
)

// DiscoveryController represents the controller.
type DiscoveryController struct {
	kubeClient       kubernetes.Interface
	serviceQueue     workqueue.RateLimitingInterface
	ingressQueue     workqueue.RateLimitingInterface
	informerFactory  informers.SharedInformerFactory
	serviceInformer  cache.SharedInformer
	endpointInformer cache.SharedInformer
	ingressInformer  cache.SharedInformer
	webclient        *odinrest.Webclient
	deleteIndexer    cache.Indexer
	namespace        string
}

// NewDiscoveryController creates a new DiscoveryController.
func NewDiscoveryController(kubeConfig *rest.Config, namespace string, webclient *odinrest.Webclient) (*DiscoveryController, error) {
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	informerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, 0, informers.WithNamespace(namespace))
	serviceInformer := informerFactory.Core().V1().Services().Informer()
	ingressInformer := informerFactory.Networking().V1().Ingresses().Informer()
	endpointInformer := informerFactory.Core().V1().Endpoints().Informer()
	deleteIndexer := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, cache.Indexers{})

	controller := &DiscoveryController{
		kubeClient:       kubeClient,
		serviceQueue:     workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		ingressQueue:     workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		informerFactory:  informerFactory,
		serviceInformer:  serviceInformer,
		endpointInformer: endpointInformer,
		ingressInformer:  ingressInformer,
		webclient:        webclient,
		deleteIndexer:    deleteIndexer,
		namespace:        namespace,
	}

	return controller, nil
}

// difference Helper function to find the difference between two slices
func getRecordsToDelete(oldRecords []string, newRecords []string) []string {
	m := make(map[string]bool)
	for _, item := range newRecords {
		m[item] = true
	}
	var diff []string
	for _, item := range oldRecords {
		if !m[item] {
			diff = append(diff, item)
		}
	}
	return diff
}

// Run starts the DiscoveryController.
func (c *DiscoveryController) Run(threadiness int, stopCh <-chan struct{}) {
	defer c.serviceQueue.ShutDown()

	// Set up event handlers for informers
	c.serviceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			annotation, exists := obj.(*corev1.Service).Annotations[odinDiscoveryAddressAnnotationKey]
			if exists {
				c.handleObject(obj.(*corev1.Service), annotation, CreateResource)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			/* there can be a number of cases:
			1. Old object doesn't have the odinDiscoveryAddressAnnotationKey annotation but new object does ==> create dns of new obj
			2. Old object has the odinDiscoveryAddressAnnotationKey annotation but new object doesn't ==> delete dns of old obj
			3. Both have the annotation: Delete for old, create for new
			4. Both don't have the annotation: nothing to do
			*/

			oldObjAnnotation, oldObjAnnotationExists := oldObj.(*corev1.Service).Annotations[odinDiscoveryAddressAnnotationKey]
			newObjAnnotation, newObjAnnotationExists := newObj.(*corev1.Service).Annotations[odinDiscoveryAddressAnnotationKey]
			if !oldObjAnnotationExists && newObjAnnotationExists {
				c.handleObject(newObj.(*corev1.Service), newObjAnnotation, CreateResource)
			} else if oldObjAnnotationExists && !newObjAnnotationExists {
				c.handleObject(oldObj.(*corev1.Service), oldObjAnnotation, DeleteResource)
			} else if oldObjAnnotationExists && newObjAnnotationExists {
				//handle the deletion for old object directly, cannot use the queue since key for both objects is same (namespace/name)
				oldAnnotations := strings.Split(oldObjAnnotation, ",")
				newAnnotations := strings.Split(newObjAnnotation, ",")
				deleteRecords := getRecordsToDelete(oldAnnotations, newAnnotations)
				for _, ann := range deleteRecords {
					c.webclient.AddToBatch(odinrest.Request{Action: "DELETE", Record: odinrest.Record{Name: ann, Values: []string{}}})
				}
				c.handleObject(newObj.(*corev1.Service), newObjAnnotation, CreateResource)
			}
		},
		DeleteFunc: func(obj interface{}) {
			annotation, exists := obj.(*corev1.Service).Annotations[odinDiscoveryAddressAnnotationKey]
			if exists {
				c.handleObject(obj.(*corev1.Service), annotation, DeleteResource)
			}
		},
	})

	c.endpointInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			/* there can be a number of cases:
			1. Old object doesn't have the odinDiscoveryAddressAnnotationKey annotation but new object does ==> create dns of new obj
			2. Old object has the odinDiscoveryAddressAnnotationKey annotation but new object doesn't ==> delete dns of old obj
			3. Both have the annotation: Delete for old, create for new
			4. Both don't have the annotation: nothing to do
			*/

			oldEndpoints := oldObj.(*corev1.Endpoints)
			oldService, oerr := c.kubeClient.CoreV1().Services(oldEndpoints.Namespace).Get(context.Background(), oldEndpoints.Name, metav1.GetOptions{})

			newEndpoints := newObj.(*corev1.Endpoints)
			newService, nerr := c.kubeClient.CoreV1().Services(newEndpoints.Namespace).Get(context.Background(), newEndpoints.Name, metav1.GetOptions{})

			if oerr == nil && nerr == nil {
				oldObjAnnotation, oldObjAnnotationExists := oldService.Annotations[odinDiscoveryAddressAnnotationKey]
				newObjAnnotation, newObjAnnotationExists := newService.Annotations[odinDiscoveryAddressAnnotationKey]
				if !oldObjAnnotationExists && newObjAnnotationExists {
					c.handleObject(newService, newObjAnnotation, CreateResource)
				} else if oldObjAnnotationExists && !newObjAnnotationExists {
					c.handleObject(oldService, oldObjAnnotation, DeleteResource)
				} else if oldObjAnnotationExists && newObjAnnotationExists {
					oldAnnotations := strings.Split(oldObjAnnotation, ",")
					newAnnotations := strings.Split(newObjAnnotation, ",")
					deleteRecords := getRecordsToDelete(oldAnnotations, newAnnotations)
					for _, ann := range deleteRecords {
						c.webclient.AddToBatch(odinrest.Request{Action: "DELETE", Record: odinrest.Record{Name: ann, Values: []string{}}})
					}
					c.handleObject(newService, newObjAnnotation, CreateResource)
				}
			}
		},
	})

	c.ingressInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			annotation, exists := obj.(*networkingv1.Ingress).Annotations[odinDiscoveryAddressAnnotationKey]
			if exists {
				c.handleObject(obj.(*networkingv1.Ingress), annotation, CreateResource)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			/* there can be a number of cases:
			1. Old object doesn't have the odinDiscoveryAddressAnnotationKey annotation but new object does ==> create dns of new obj
			2. Old object has the odinDiscoveryAddressAnnotationKey annotation but new object doesn't ==> delete dns of old obj
			3. Both have the annotation: Delete for old, create for new
			4. Both don't have the annotation: nothing to do
			*/

			oldObjAnnotation, oldObjAnnotationExists := oldObj.(*networkingv1.Ingress).Annotations[odinDiscoveryAddressAnnotationKey]
			newObjAnnotation, newObjAnnotationExists := newObj.(*networkingv1.Ingress).Annotations[odinDiscoveryAddressAnnotationKey]
			if !oldObjAnnotationExists && newObjAnnotationExists {
				c.handleObject(newObj.(*networkingv1.Ingress), newObjAnnotation, CreateResource)
			} else if oldObjAnnotationExists && !newObjAnnotationExists {
				c.handleObject(oldObj.(*networkingv1.Ingress), oldObjAnnotation, DeleteResource)
			} else if oldObjAnnotationExists && newObjAnnotationExists {
				//handle the deletion for old object directly, cannot use the queue since key for both objects is same (namespace/name)
				oldAnnotations := strings.Split(oldObjAnnotation, ",")
				newAnnotations := strings.Split(newObjAnnotation, ",")
				deleteRecords := getRecordsToDelete(oldAnnotations, newAnnotations)
				for _, ann := range deleteRecords {
					c.webclient.AddToBatch(odinrest.Request{Action: "DELETE", Record: odinrest.Record{Name: ann, Values: []string{}}})
				}
				c.handleObject(newObj.(*networkingv1.Ingress), newObjAnnotation, CreateResource)
			}
		},
		DeleteFunc: func(obj interface{}) {
			annotation, exists := obj.(*networkingv1.Ingress).Annotations[odinDiscoveryAddressAnnotationKey]
			if exists {
				c.handleObject(obj.(*networkingv1.Ingress), annotation, DeleteResource)
			}
		},
	})

	// Start informers
	c.informerFactory.Start(stopCh)

	// Wait for the informers' cache to be synced before processing events
	if !cache.WaitForCacheSync(stopCh, c.serviceInformer.HasSynced, c.ingressInformer.HasSynced) {
		log.Error("failed to sync informers")
		return
	}

	// Start workers to process the work queue
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runServiceWorker, time.Second, stopCh)
		go wait.Until(c.runIngressWorker, time.Second, stopCh)
	}

	// Wait until the controller is told to stop
	<-stopCh
}

// resource can be either *corev1.Service or *networkingv1.Interface
func (c *DiscoveryController) handleObject(resource interface{}, annotationValue string, action int) {
	// Annotation is present, add the resource key (e.g., "namespace/name") to the work queue
	var queue workqueue.RateLimitingInterface
	switch r := resource.(type) {
	case *corev1.Service:
		log.Infof("Service %s has the %s annotation with value: %s\n", r.Name, odinDiscoveryAddressAnnotationKey, annotationValue)
		queue = c.serviceQueue
	case *networkingv1.Ingress:
		log.Infof("Ingress %s has the %s annotation with value: %s\n", r.Name, odinDiscoveryAddressAnnotationKey, annotationValue)
		queue = c.ingressQueue
	}

	switch action {
	case CreateResource:
		var key string
		var err error
		if key, err = cache.MetaNamespaceKeyFunc(resource); err != nil {
			utilruntime.HandleError(err)
			return
		}
		queue.Add(key)
	case DeleteResource:
		var key string
		var err error
		if key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(resource); err != nil {
			utilruntime.HandleError(err)
			return
		}
		c.deleteIndexer.Add(resource)
		queue.Add(key)
	}
}

func (c *DiscoveryController) runServiceWorker() {
	for c.processNextItem(corev1.Service{}) {
	}
}

func (c *DiscoveryController) runIngressWorker() {
	for c.processNextItem(networkingv1.Ingress{}) {
	}
}

func (c *DiscoveryController) processNextItem(resourceObject interface{}) bool {
	switch resource := resourceObject.(type) {
	case corev1.Service:
		return c.processItem(c.serviceQueue, c.serviceInformer)
	case networkingv1.Ingress:
		return c.processItem(c.ingressQueue, c.ingressInformer)
	default:
		log.Info("Unsupported resource type", resource)
		return true //don't stop for random resources
	}
}

func (c *DiscoveryController) processItem(queue workqueue.RateLimitingInterface, informer cache.SharedInformer) bool {
	// Get the next item from the work queue
	obj, shutdown := queue.Get()
	if shutdown {
		return false // Stop processing if the controller is shutting down
	}
	defer queue.Done(obj)

	var key string
	var ok bool
	// We expect strings to come off the respective queue. These are of the
	// form namespace/name. We do this as the delayed nature of the
	// queue means the items in the informer cache may actually be
	// more up to date that when the item was initially put onto the
	// queue.
	if key, ok = obj.(string); !ok {
		// As the item in the queue is actually invalid, we call
		// Forget here else we'd go into a loop of attempting to
		// process a work item that is invalid.
		queue.Forget(obj)
		utilruntime.HandleError(fmt.Errorf("expected string in queue but got %#v", obj))
		return true
	}

	// Check if the resource with the key still exists in the cache refer https://github.com/kubernetes/client-go/issues/297
	resource, exists, err := informer.GetStore().GetByKey(key)

	if err != nil {
		log.Errorf("fetching resource %s from cache failed with error: %s", resource, err.Error())
		queue.Forget(obj)
		return true
	}

	if !exists {
		c.handleDeletion(key)
	} else {
		c.actOnResource(resource, CreateResource)
	}

	log.Infof("Processing obj: %v\n", obj)
	queue.Forget(obj)

	return true
}

func (c *DiscoveryController) handleDeletion(key string) {
	deletedObj, exists, err := c.deleteIndexer.GetByKey(key)
	if err != nil || !exists {
		log.Errorf("deleting resource %s failed with error: %v", key, err)
		return
	}

	c.actOnResource(deletedObj, DeleteResource)
	c.deleteIndexer.Delete(key)
}

func (c *DiscoveryController) actOnResource(resource interface{}, action int) {
	switch resourceType := resource.(type) {
	case *corev1.Service:
		c.processService(resource.(*corev1.Service), action)
	case *networkingv1.Ingress:
		c.processIngress(resource.(*networkingv1.Ingress), action)
	default:
		log.Errorf("unsupported resource type %v for action %d", resourceType, action)
	}
}

func extractPodIPs(endpoints *corev1.Endpoints) []string {
	var podIPs []string
	for _, subset := range endpoints.Subsets {
		for _, address := range subset.Addresses {
			podIPs = append(podIPs, address.IP)
		}
	}
	return podIPs
}

func determineServiceAction(action int, podIPs []string, service *corev1.Service) string {
	switch action {
	case CreateResource:
		if len(podIPs) == 0 {
			log.Infof("No pods behind endpoint, deleting route for service: %v\n", service)
			return "DELETE"
		}
		log.Infof("Creating route for service: %v\n", service)
		return "UPSERT"
	case DeleteResource:
		log.Infof("Deleting route for service: %v\n", service)
		return "DELETE"
	default:
		log.Error("unknown action on service")
		return ""
	}
}

func (c *DiscoveryController) processService(service *corev1.Service, action int) {
	log.Infof("Processing service: %v\n", service)

	// Check if the service has endpoints
	if service.Spec.ClusterIP != "" && len(service.Spec.Ports) > 0 {
		endpoints, err := c.kubeClient.CoreV1().Endpoints(service.ObjectMeta.Namespace).Get(context.Background(), service.Name, metav1.GetOptions{})
		if err != nil {
			log.Errorf("Error getting endpoints for service %s: %v\n", service.Name, err)
		}
		podIPs := extractPodIPs(endpoints)

		annotation := service.Annotations[odinDiscoveryAddressAnnotationKey]
		annotations := strings.Split(annotation, ",")
		for _, ann := range annotations {
			c.webclient.AddToBatch(odinrest.Request{Action: determineServiceAction(action, podIPs, service), Record: odinrest.Record{Name: ann, Values: podIPs}})
		}
	}
}

func (c *DiscoveryController) processIngress(ingress *networkingv1.Ingress, action int) {
	log.Infof("Processing ingress: %v\n", ingress)

	var addresses []string
	if len(ingress.Status.LoadBalancer.Ingress) > 0 {
		for _, lb := range ingress.Status.LoadBalancer.Ingress {
			if lb.IP != "" {
				addresses = append(addresses, lb.IP)
			} else if lb.Hostname != "" {
				addresses = append(addresses, lb.Hostname)
			}
		}
	} else {
		log.Warnf("Ingress %s/%s has no external address yet", ingress.Namespace, ingress.Name)
		// Would be handled when the ingress is updated (with the address) now
		return

	}
	annotation := ingress.Annotations[odinDiscoveryAddressAnnotationKey]

	var resourceAction string
	if action == CreateResource {
		log.Infof("Creating route for ingress: %v\n", ingress)
		resourceAction = "UPSERT"
	} else if action == DeleteResource {
		log.Infof("Deleting route for ingress: %v\n", ingress)
		resourceAction = "DELETE"
	} else {
		log.Error("unknown action on ingress")
	}
	annotations := strings.Split(annotation, ",")
	for _, ann := range annotations {
		c.webclient.AddToBatch(odinrest.Request{Action: resourceAction, Record: odinrest.Record{Name: ann, Values: addresses}})
	}
}
