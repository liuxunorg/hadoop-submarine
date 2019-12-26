/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package controller

import (
	"fmt"
	"github.com/apache/submarine/submarine-cloud/pkg/controller/pod"
	"github.com/apache/submarine/submarine-cloud/pkg/controller/sanitycheck"
	"github.com/apache/submarine/submarine-cloud/pkg/submarine"
	"github.com/golang/glog"
	apiv1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	policyv1listers "k8s.io/client-go/listers/policy/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"math"
	"reflect"
	"time"

	rapi "github.com/apache/submarine/submarine-cloud/pkg/apis/submarine/v1alpha1"
	sClient "github.com/apache/submarine/submarine-cloud/pkg/client/clientset/versioned"
	sInformers "github.com/apache/submarine/submarine-cloud/pkg/client/informers/externalversions"
	sListers "github.com/apache/submarine/submarine-cloud/pkg/client/listers/submarine/v1alpha1"
	clientset "k8s.io/client-go/kubernetes"
)

// Controller contains all controller fields
type Controller struct {
	kubeClient      clientset.Interface
	submarineClient sClient.Interface

	submarineClusterLister sListers.SubmarineClusterLister
	SubmarineClusterSynced cache.InformerSynced

	podLister corev1listers.PodLister
	PodSynced cache.InformerSynced

	serviceLister corev1listers.ServiceLister
	ServiceSynced cache.InformerSynced

	podDisruptionBudgetLister  policyv1listers.PodDisruptionBudgetLister
	PodDiscruptionBudgetSynced cache.InformerSynced

	podControl                 pod.SubmarineClusterControlInteface
	serviceControl             ServicesControlInterface
	podDisruptionBudgetControl PodDisruptionBudgetsControlInterface

	updateHandler func(cluster *rapi.SubmarineCluster) (*rapi.SubmarineCluster, error) // callback to update SubmarineCluster. Added as member for testing

	queue    workqueue.RateLimitingInterface // SubmarineClusters to be synced
	recorder record.EventRecorder

	config *Config
}

// NewController builds and return new controller instance
func NewController(cfg *Config, kubeClient clientset.Interface, submarineClient sClient.Interface, kubeInformer kubeinformers.SharedInformerFactory, rInformer sInformers.SharedInformerFactory) *Controller {
	glog.Info("NewController()")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})

	serviceInformer := kubeInformer.Core().V1().Services()
	podInformer := kubeInformer.Core().V1().Pods()
	submarineInformer := rInformer.Submarine().V1alpha1().SubmarineClusters()
	podDisruptionBudgetInformer := kubeInformer.Policy().V1beta1().PodDisruptionBudgets()

	ctrl := &Controller{
		kubeClient:                 kubeClient,
		submarineClient:            submarineClient,
		submarineClusterLister:     submarineInformer.Lister(),
		SubmarineClusterSynced:     submarineInformer.Informer().HasSynced,
		podLister:                  podInformer.Lister(),
		PodSynced:                  podInformer.Informer().HasSynced,
		serviceLister:              serviceInformer.Lister(),
		ServiceSynced:              serviceInformer.Informer().HasSynced,
		podDisruptionBudgetLister:  podDisruptionBudgetInformer.Lister(),
		PodDiscruptionBudgetSynced: podDisruptionBudgetInformer.Informer().HasSynced,

		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "submarinecluster"),
		recorder: eventBroadcaster.NewRecorder(scheme.Scheme, apiv1.EventSource{Component: "submarinecluster-controller"}),

		config: cfg,
	}

	submarineInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.onAddSubmarineCluster,
			UpdateFunc: ctrl.onUpdateSubmarineCluster,
			DeleteFunc: ctrl.onDeleteSubmarineCluster,
		},
	)

	podInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.onAddPod,
			UpdateFunc: ctrl.onUpdatePod,
			DeleteFunc: ctrl.onDeletePod,
		},
	)

	ctrl.updateHandler = ctrl.updateSubmarineCluster
	ctrl.podControl = pod.NewSubmarineClusterControl(ctrl.podLister, ctrl.kubeClient, ctrl.recorder)
	ctrl.serviceControl = NewServicesControl(ctrl.kubeClient, ctrl.recorder)
	ctrl.podDisruptionBudgetControl = NewPodDisruptionBudgetsControl(ctrl.kubeClient, ctrl.recorder)

	return ctrl
}

// Run executes the Controller
func (c *Controller) Run(stop <-chan struct{}) error {
	glog.Infof("Starting SubmarineCluster controller")

	if !cache.WaitForCacheSync(stop, c.PodSynced, c.SubmarineClusterSynced, c.ServiceSynced) {
		return fmt.Errorf("Timed out waiting for caches to sync")
	}

	for i := 0; i < c.config.NbWorker; i++ {
		go wait.Until(c.runWorker, time.Second, stop)
	}

	<-stop
	return nil
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

func (c *Controller) processNextItem() bool {
	glog.Infof("processNextItem")
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	needRequeue, err := c.sync(key.(string))
	if err == nil {
		c.queue.Forget(key)
	} else {
		utilruntime.HandleError(fmt.Errorf("Error syncing submarinecluster: %v", err))
		c.queue.AddRateLimited(key)
		return true
	}

	if needRequeue {
		glog.V(4).Info("processNextItem: Requeue key:", key)
		c.queue.AddRateLimited(key)
	}

	return true
}

func (c *Controller) sync(key string) (bool, error) {
	glog.V(2).Infof("sync() key:%s", key)
	startTime := metav1.Now()
	defer func() {
		glog.V(2).Infof("Finished syncing SubmarineCluster %q (%v", key, time.Since(startTime.Time))
	}()
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return false, err
	}
	glog.V(6).Infof("Syncing %s/%s", namespace, name)
	sharedSubmarineCluster, err := c.submarineClusterLister.SubmarineClusters(namespace).Get(name)
	if err != nil {
		glog.Errorf("unable to get SubmarineCluster %s/%s: %v. Maybe deleted", namespace, name, err)
		return false, nil
	}

	if !rapi.IsDefaultedSubmarineCluster(sharedSubmarineCluster) {
		defaultedSubmarineCluster := rapi.DefaultSubmarineCluster(sharedSubmarineCluster)
		if _, err = c.updateHandler(defaultedSubmarineCluster); err != nil {
			glog.Errorf("SubmarineCluster %s/%s updated error:, err", namespace, name)
			return false, fmt.Errorf("unable to default SubmarineCluster %s/%s: %v", namespace, name, err)
		}
		glog.V(6).Infof("SubmarineCluster-Operator.sync Defaulted %s/%s", namespace, name)
		return false, nil
	}

	// TODO add validation

	// TODO: add test the case of graceful deletion
	if sharedSubmarineCluster.DeletionTimestamp != nil {
		return false, nil
	}

	submarinecluster := sharedSubmarineCluster.DeepCopy()

	// Init status.StartTime
	if submarinecluster.Status.StartTime == nil {
		submarinecluster.Status.StartTime = &startTime
		if _, err := c.updateHandler(submarinecluster); err != nil {
			glog.Errorf("SubmarineCluster %s/%s: unable init startTime: %v", namespace, name, err)
			return false, nil
		}
		glog.V(4).Infof("SubmarineCluster %s/%s: startTime updated", namespace, name)
		return false, nil
	}
	return c.syncCluster(submarinecluster)
}

func (c *Controller) syncCluster(submarineCluster *rapi.SubmarineCluster) (forceRequeue bool, err error) {
	glog.V(1).Info("syncCluster START")
	defer glog.V(1).Info("syncCluster STOP")
	forceRequeue = false
	redisClusterService, err := c.getRedisClusterService(submarineCluster)
	if err != nil {
		glog.Errorf("SubmarineCluster-Operator.sync unable to retrieves service associated to the SubmarineCluster: %s/%s", submarineCluster.Namespace, submarineCluster.Name)
		return forceRequeue, err
	}
	if redisClusterService == nil {
		if _, err = c.serviceControl.CreateSubmarineClusterService(submarineCluster); err != nil {
			glog.Errorf("SubmarineCluster-Operator.sync unable to create service associated to the SubmarineCluster: %s/%s", submarineCluster.Namespace, submarineCluster.Name)
			return forceRequeue, err
		}
	}

	redisClusterPodDisruptionBudget, err := c.getRedisClusterPodDisruptionBudget(submarineCluster)
	if err != nil {
		glog.Errorf("SubmarineCluster-Operator.sync unable to retrieves podDisruptionBudget associated to the SubmarineCluster: %s/%s", submarineCluster.Namespace, submarineCluster.Name)
		return forceRequeue, err
	}
	if redisClusterPodDisruptionBudget == nil {
		if _, err = c.podDisruptionBudgetControl.CreateSubmarineClusterPodDisruptionBudget(submarineCluster); err != nil {
			glog.Errorf("SubmarineCluster-Operator.sync unable to create podDisruptionBudget associated to the SubmarineCluster: %s/%s", submarineCluster.Namespace, submarineCluster.Name)
			return forceRequeue, err
		}
	}

	redisClusterPods, err := c.podControl.GetSubmarineClusterPods(submarineCluster)
	if err != nil {
		glog.Errorf("SubmarineCluster-Operator.sync unable to retrieves pod associated to the SubmarineCluster: %s/%s", submarineCluster.Namespace, submarineCluster.Name)
		return forceRequeue, err
	}

	Pods, LostPods := filterLostNodes(redisClusterPods)
	if len(LostPods) != 0 {
		for _, p := range LostPods {
			err := c.podControl.DeletePodNow(submarineCluster, p.Name)
			glog.Errorf("Lost node with pod %s. Deleting... %v", p.Name, err)
		}
		redisClusterPods = Pods
	}

	// RedisAdmin is used access the Redis process in the different pods.
	admin, err := NewRedisAdmin(redisClusterPods, &c.config.submarine)
	if err != nil {
		return forceRequeue, fmt.Errorf("unable to create the redis.Admin, err:%v", err)
	}
	defer admin.Close()

	clusterInfos, errGetInfos := admin.GetClusterInfos()
	if errGetInfos != nil {
		glog.Errorf("Error when get cluster infos to rebuild bom : %v", errGetInfos)
		if clusterInfos.Status == submarine.ClusterInfosPartial {
			return false, fmt.Errorf("partial Cluster infos")
		}
	}

	// From the Redis cluster nodes connections, build the cluster status
	// Calculate the actual cluster status through node information, cluster Pod list, and CR
	// The cluster status includes: whether it is normal, the number of Ready Pods, the number of Masters,
	// the number of Redis instances in operation, the list of Redis instances, replication factors, etc.
	clusterStatus, err := c.buildClusterStatus(clusterInfos, redisClusterPods)
	if err != nil {
		glog.Errorf("unable to build the RedisClusterStatus, err:%v", err)
		return forceRequeue, fmt.Errorf("unable to build clusterStatus, err:%v", err)
	}

	// If the cluster status (Status.Cluster) in the CR does not match the actual situation, update
	updated, err := c.updateClusterIfNeed(submarineCluster, clusterStatus)
	if err != nil {
		return forceRequeue, err
	}
	if updated {
		// If the cluster status changes requeue the key. Because we want to apply Redis Cluster operation only on stable cluster,
		// already stored in the API server.
		glog.V(3).Infof("cluster updated %s-%s", submarineCluster.Namespace, submarineCluster.Name)
		forceRequeue = true
		return forceRequeue, nil
	}

	// If the CR state matches the actual state of the Redis cluster, then check if reconciliation is required-let the actual state match the expected state
	allPodsNotReady := true
	if (clusterStatus.NbPods - clusterStatus.NbRedisRunning) != 0 {
		glog.V(3).Infof("All pods not ready wait to be ready, nbPods: %d, nbPodsReady: %d", clusterStatus.NbPods, clusterStatus.NbRedisRunning)
		allPodsNotReady = false
	}

	// Now check if the Operator need to execute some operation the redis cluster. if yes run the clusterAction(...) method.
	needSanitize, err := c.checkSanityCheck(submarineCluster, admin, clusterInfos)
	if err != nil {
		glog.Errorf("checkSanityCheck, error happened in dryrun mode, err:%v", err)
		return false, err
	}

	// If all Pods are not ready and need rolling updates (Pod and PodTemplate do not match), more or fewer Pods are needed,
	// or the number of master nodes and replication factor are incorrect
	// Or, need to perform "clean up"
	// Then, perform Redis cluster management operations to approximate the expected state and update the status of RedisCluster
	if (allPodsNotReady && needClusterOperation(submarineCluster)) || needSanitize {
		var requeue bool
		forceRequeue = false
		// Perform cluster management operations, including creating / deleting pods and configuring Redis
		requeue, err = c.clusterAction(admin, submarineCluster, clusterInfos)
		if err != nil {
			glog.Errorf("error during action on cluster: %s-%s, err: %v", submarineCluster.Namespace, submarineCluster.Name, err)
		} else if requeue {
			forceRequeue = true
		}
		_, err = c.updateRedisCluster(submarineCluster)
		return forceRequeue, err
	}

	// Reset all conditions and reconcile
	if setRebalancingCondition(&submarineCluster.Status, false) ||
		setRollingUpdategCondition(&submarineCluster.Status, false) ||
		setScalingCondition(&submarineCluster.Status, false) ||
		setClusterStatusCondition(&submarineCluster.Status, true) {
		_, err = c.updateHandler(submarineCluster)
		return forceRequeue, err
	}

	return false, nil
}

func (c *Controller) onAddSubmarineCluster(obj interface{}) {
	glog.Infof("onAddSubmarineCluster(%v)", obj)
	rediscluster, ok := obj.(*rapi.SubmarineCluster)
	if !ok {
		glog.Errorf("adding RedisCluster, expected RedisCluster object. Got: %+v", obj)
		return
	}
	glog.V(6).Infof("onAddRedisCluster %s/%s", rediscluster.Namespace, rediscluster.Name)
	if !reflect.DeepEqual(rediscluster.Status, rapi.SubmarineClusterStatus{}) {
		glog.Errorf("rediscluster %s/%s created with non empty status. Going to be removed", rediscluster.Namespace, rediscluster.Name)

		if _, err := cache.MetaNamespaceKeyFunc(rediscluster); err != nil {
			glog.Errorf("couldn't get key for RedisCluster (to be deleted) %s/%s: %v", rediscluster.Namespace, rediscluster.Name, err)
			return
		}
		// TODO: how to remove a rediscluster created with an invalid or even with a valid status. What in case of error for this delete?
		if err := c.deleteRedisCluster(rediscluster.Namespace, rediscluster.Name); err != nil {
			glog.Errorf("unable to delete non empty status RedisCluster %s/%s: %v. No retry will be performed.", rediscluster.Namespace, rediscluster.Name, err)
		}

		return
	}

	c.enqueue(rediscluster)
}

func (c *Controller) onDeleteSubmarineCluster(obj interface{}) {
	glog.Infof("onDeleteSubmarineCluster(%v)", obj)
}

func (c *Controller) onUpdateSubmarineCluster(oldObj, newObj interface{}) {
	glog.Infof("onUpdateSubmarineCluster(%v, %v)", oldObj, newObj)

	rediscluster, ok := newObj.(*rapi.SubmarineCluster)
	if !ok {
		glog.Errorf("Expected RedisCluster object. Got: %+v", newObj)
		return
	}
	glog.V(6).Infof("onUpdateRedisCluster %s/%s", rediscluster.Namespace, rediscluster.Name)
	c.enqueue(rediscluster)
}

func (c *Controller) onAddPod(obj interface{}) {
	glog.Infof("onAddPod()")
	pod, ok := obj.(*apiv1.Pod)
	if !ok {
		glog.Errorf("adding Pod, expected Pod object. Got: %+v", obj)
		return
	}
	if _, ok := pod.GetObjectMeta().GetLabels()[rapi.ClusterNameLabelKey]; !ok {
		return
	}
	redisCluster, err := c.getRedisClusterFromPod(pod)
	if err != nil {
		glog.Errorf("unable to retrieve the associated rediscluster for pod %s/%s:%v", pod.Namespace, pod.Name, err)
		return
	}
	if redisCluster == nil {
		glog.Errorf("empty redisCluster. Unable to retrieve the associated rediscluster for the pod  %s/%s", pod.Namespace, pod.Name)
		return
	}

	c.enqueue(redisCluster)
}

func (c *Controller) onUpdatePod(oldObj, newObj interface{}) {
	glog.Infof("onUpdatePod()")
	oldPod := oldObj.(*apiv1.Pod)
	newPod := newObj.(*apiv1.Pod)
	if oldPod.ResourceVersion == newPod.ResourceVersion { // Since periodic resync will send update events for all known Pods.
		return
	}
	if _, ok := newPod.GetObjectMeta().GetLabels()[rapi.ClusterNameLabelKey]; !ok {
		return
	}
	glog.V(6).Infof("onUpdatePod old=%v, cur=%v ", oldPod.Name, newPod.Name)
	redisCluster, err := c.getRedisClusterFromPod(newPod)
	if err != nil {
		glog.Errorf("RedisCluster-Operator.onUpdateJob cannot get redisclusters for Pod %s/%s: %v", newPod.Namespace, newPod.Name, err)
		return
	}
	if redisCluster == nil {
		glog.Errorf("empty redisCluster .onUpdateJob cannot get redisclusters for Pod %s/%s", newPod.Namespace, newPod.Name)
		return
	}

	c.enqueue(redisCluster)

	// TODO: in case of relabelling ?
	// TODO: in case of labelSelector relabelling?
}

func (c *Controller) onDeletePod(obj interface{}) {
	glog.Infof("onDeletePod()")
	pod, ok := obj.(*apiv1.Pod)
	if _, ok := pod.GetObjectMeta().GetLabels()[rapi.ClusterNameLabelKey]; !ok {
		return
	}
	glog.V(6).Infof("onDeletePod old=%v", pod.Name)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			glog.Errorf("Couldn't get object from tombstone %+v", obj)
			return
		}
		pod, ok = tombstone.Obj.(*apiv1.Pod)
		if !ok {
			glog.Errorf("Tombstone contained object that is not a pod %+v", obj)
			return
		}
	}

	redisCluster, err := c.getRedisClusterFromPod(pod)
	if err != nil {
		glog.Errorf("RedisCluster-Operator.onDeletePod: %v", err)
		return
	}
	if redisCluster == nil {
		glog.Errorf("empty redisCluster . RedisCluster-Operator.onDeletePod")
		return
	}

	c.enqueue(redisCluster)
}

func (c *Controller) updateSubmarineCluster(submarineCluster *rapi.SubmarineCluster) (*rapi.SubmarineCluster, error) {
	rc, err := c.submarineClient.SubmarineV1alpha1().SubmarineClusters(submarineCluster.Namespace).Update(submarineCluster)
	if err != nil {
		glog.Errorf("updateSubmarineCluster cluster: [%v] error: %v", *submarineCluster, err)
		return rc, err
	}

	glog.V(6).Infof("SubmarineCluster %s/%s updated", submarineCluster.Namespace, submarineCluster.Name)
	return rc, nil
}

// enqueue adds key in the controller queue
func (c *Controller) enqueue(rediscluster *rapi.SubmarineCluster) {
	key, err := cache.MetaNamespaceKeyFunc(rediscluster)
	if err != nil {
		glog.Errorf("RedisCluster-Controller:enqueue: couldn't get key for RedisCluster %s/%s: %v", rediscluster.Namespace, rediscluster.Name, err)
		return
	}
	c.queue.Add(key)
}

func (c *Controller) getRedisClusterService(redisCluster *rapi.SubmarineCluster) (*apiv1.Service, error) {
	serviceName := getServiceName(redisCluster)
	labels, err := pod.GetLabelsSet(redisCluster)
	if err != nil {
		return nil, fmt.Errorf("couldn't get cluster label, err: %v ", err)
	}

	svcList, err := c.serviceLister.Services(redisCluster.Namespace).List(labels.AsSelector())
	if err != nil {
		return nil, fmt.Errorf("couldn't list service with label:%s, err:%v ", labels.String(), err)
	}
	var svc *apiv1.Service
	for i, s := range svcList {
		if s.Name == serviceName {
			svc = svcList[i]
		}
	}
	return svc, nil
}

func (c *Controller) getRedisClusterPodDisruptionBudget(redisCluster *rapi.SubmarineCluster) (*policyv1.PodDisruptionBudget, error) {
	podDisruptionBudgetName := redisCluster.Name
	labels, err := pod.GetLabelsSet(redisCluster)
	if err != nil {
		return nil, fmt.Errorf("couldn't get cluster label, err: %v ", err)
	}

	pdbList, err := c.podDisruptionBudgetLister.PodDisruptionBudgets(redisCluster.Namespace).List(labels.AsSelector())
	if err != nil {
		return nil, fmt.Errorf("couldn't list PodDisruptionBudget with label:%s, err:%v ", labels.String(), err)
	}
	var pdb *policyv1.PodDisruptionBudget
	for i, p := range pdbList {
		if p.Name == podDisruptionBudgetName {
			pdb = pdbList[i]
		}
	}
	return pdb, nil
}

func (c *Controller) buildClusterStatus(clusterInfos *submarine.ClusterInfos, pods []*apiv1.Pod) (*rapi.SubmarineClusterClusterStatus, error) {
	clusterStatus := &rapi.SubmarineClusterClusterStatus{}
	clusterStatus.NbPodsReady = 0
	clusterStatus.NbRedisRunning = 0
	clusterStatus.MaxReplicationFactor = 0
	clusterStatus.MinReplicationFactor = 0

	clusterStatus.NbPods = int32(len(pods))
	var nbRedisRunning, nbPodsReady int32

	nbMaster := int32(0)
	nbSlaveByMaster := map[string]int{}

	for _, pod := range pods {
		if podready, _ := IsPodReady(pod); podready {
			nbPodsReady++
		}

		newNode := rapi.SubmarineClusterNode{
			PodName: pod.Name,
			IP:      pod.Status.PodIP,
			Pod:     pod,
			Slots:   []string{},
		}
		// find corresponding Redis node
		redisNodes, err := clusterInfos.GetNodes().GetNodesByFunc(func(node *submarine.Node) bool {
			return node.IP == pod.Status.PodIP
		})
		if err != nil {
			glog.Errorf("Unable to retrieve the associated Redis Node with the pod: %s, ip:%s, err:%v", pod.Name, pod.Status.PodIP, err)
			continue
		}
		if len(redisNodes) == 1 {
			redisNode := redisNodes[0]
			if submarine.IsMasterWithSlot(redisNode) {
				if _, ok := nbSlaveByMaster[redisNode.ID]; !ok {
					nbSlaveByMaster[redisNode.ID] = 0
				}
				nbMaster++
			}

			newNode.ID = redisNode.ID
			newNode.Role = redisNode.GetRole()
			newNode.Port = redisNode.Port
			newNode.Slots = []string{}
			if submarine.IsSlave(redisNode) && redisNode.MasterReferent != "" {
				nbSlaveByMaster[redisNode.MasterReferent] = nbSlaveByMaster[redisNode.MasterReferent] + 1
				newNode.MasterRef = redisNode.MasterReferent
			}
			///if len(redisNode.Slots) > 0 {
			///	slots := redis.SlotRangesFromSlots(redisNode.Slots)
			///	for _, slot := range slots {
			///		newNode.Slots = append(newNode.Slots, slot.String())
			///	}
			///}
			nbRedisRunning++
		}
		clusterStatus.Nodes = append(clusterStatus.Nodes, newNode)
	}
	clusterStatus.NbRedisRunning = nbRedisRunning
	clusterStatus.NumberOfMaster = nbMaster
	clusterStatus.NbPodsReady = nbPodsReady
	clusterStatus.Status = rapi.ClusterStatusOK

	minReplicationFactor := math.MaxInt32
	maxReplicationFactor := 0
	for _, counter := range nbSlaveByMaster {
		if counter > maxReplicationFactor {
			maxReplicationFactor = counter
		}
		if counter < minReplicationFactor {
			minReplicationFactor = counter
		}
	}
	if len(nbSlaveByMaster) == 0 {
		minReplicationFactor = 0
	}
	clusterStatus.MaxReplicationFactor = int32(maxReplicationFactor)
	clusterStatus.MinReplicationFactor = int32(minReplicationFactor)

	glog.V(3).Infof("Build Bom, current Node list : %s ", clusterStatus.String())

	return clusterStatus, nil
}

func (c *Controller) updateClusterIfNeed(cluster *rapi.SubmarineCluster, newStatus *rapi.SubmarineClusterClusterStatus) (bool, error) {
	if compareStatus(&cluster.Status.Cluster, newStatus) {
		glog.V(3).Infof("Status changed for cluster: %s-%s", cluster.Namespace, cluster.Name)
		// the status have been update, needs to update the RedisCluster
		cluster.Status.Cluster = *newStatus
		_, err := c.updateRedisCluster(cluster)
		return true, err
	}
	// TODO improve this by checking properly the kapi.Pod informations inside each Node
	cluster.Status.Cluster.Nodes = newStatus.Nodes
	return false, nil
}

func (c *Controller) updateRedisCluster(rediscluster *rapi.SubmarineCluster) (*rapi.SubmarineCluster, error) {
	rc, err := c.submarineClient.SubmarineV1alpha1().SubmarineClusters(rediscluster.Namespace).Update(rediscluster)
	if err != nil {
		glog.Errorf("updateRedisCluster cluster: [%v] error: %v", *rediscluster, err)
		return rc, err
	}

	glog.V(6).Infof("RedisCluster %s/%s updated", rediscluster.Namespace, rediscluster.Name)
	return rc, nil
}

func (c *Controller) checkSanityCheck(cluster *rapi.SubmarineCluster, admin submarine.AdminInterface, infos *submarine.ClusterInfos) (bool, error) {
	return sanitycheck.RunSanityChecks(admin, &c.config.submarine, c.podControl, cluster, infos, true)
}

func (c *Controller) deleteRedisCluster(namespace, name string) error {
	return nil
}

func (c *Controller) getRedisClusterFromPod(pod *apiv1.Pod) (*rapi.SubmarineCluster, error) {
	if len(pod.Labels) == 0 {
		return nil, fmt.Errorf("no rediscluster found for pod. Pod %s/%s has no labels", pod.Namespace, pod.Name)
	}

	clusterName, ok := pod.Labels[rapi.ClusterNameLabelKey]
	if !ok {
		return nil, fmt.Errorf("no rediscluster name found for pod. Pod %s/%s has no labels %s", pod.Namespace, pod.Name, rapi.ClusterNameLabelKey)
	}
	return c.submarineClusterLister.SubmarineClusters(pod.Namespace).Get(clusterName)
}

