package nep

import (
	"fmt"
	"github.com/blang/semver"
	"k8s.io/apimachinery/pkg/api/resource"
	"sync"
	"time"

	"git.code.oa.com/tke/log"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"git.code.oa.com/tke/api/platform/v1"
	"git.code.oa.com/tke/tke/pkg/registry/helper"
	coreV1 "k8s.io/api/core/v1"
)

const conditionTypeHealthCheck = "HealthCheck"
const conditionTypeSyncVersion = "SyncVersion"
const reasonHealthCheckFail = "HealthCheckFail"

type clusterHealth struct {
	mu         sync.Mutex
	clusterMap map[string]*v1.Cluster
}

func (s *clusterHealth) Exist(clusterName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.clusterMap[clusterName]
	return ok
}

func (s *clusterHealth) Set(cluster *v1.Cluster) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clusterMap[cluster.Name] = cluster
}

func (s *clusterHealth) Del(clusterName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clusterMap, clusterName)
}

func (c *Controller) startClusterHealthCheck(key string, cluster *v1.Cluster) {
	if !c.health.Exist(key) {
		c.health.Set(cluster)
		go wait.PollImmediateUntil(1*time.Minute, c.watchClusterHealth(cluster.Name), c.stopCh)
		log.Info("Cluster phase start new health check", log.String("clusterName", key), log.String("phase", string(cluster.Status.Phase)))
	} else {
		log.Info("Cluster phase health check exit", log.String("clusterName", key), log.String("phase", string(cluster.Status.Phase)))
	}
}

func (c *Controller) checkClusterHealth(cluster *v1.Cluster) error {
	log.Debug("begin to check cluster health", log.String("cluster", cluster.Name))
	// get cluster client
	kubeClient, err := helper.BuildExternalClientSetNoStatus(cluster, c.client.AuthenticationV1(), c.apiPublicAddress, c.debug)
	if err != nil {
		cluster.Status.Phase = v1.ClusterFailed
		cluster.Status.Message = err.Error()
		cluster.Status.Reason = reasonHealthCheckFail
		now := metaV1.Now()
		c.addOrUpdateCondition(cluster, v1.ClusterCondition{
			Type:               conditionTypeHealthCheck,
			Status:             v1.ConditionFalse,
			Message:            err.Error(),
			Reason:             reasonHealthCheckFail,
			LastTransitionTime: now,
			LastProbeTime:      now,
		})
		if err1 := c.persistUpdate(cluster); err1 != nil {
			log.Warn("Update cluster status failed", log.String("clusterName", cluster.Name), log.Err(err1))
			return err1
		}
		log.Warn("Failed to build the cluster client", log.String("clusterName", cluster.Name), log.Err(err))
		return err
	}
	//log.Debug("begin to check caclClusterResource", log.String("cluster", cluster.Name))
	//
	//res, err := c.caclClusterResource(kubeClient, cluster.Name)
	//if err != nil {
	//	cluster.Status.Phase = v1.ClusterFailed
	//	cluster.Status.Message = err.Error()
	//	cluster.Status.Reason = reasonHealthCheckFail
	//	now := metaV1.Now()
	//	c.addOrUpdateCondition(cluster, v1.ClusterCondition{
	//		Type:               conditionTypeHealthCheck,
	//		Status:             v1.ConditionFalse,
	//		Message:            err.Error(),
	//		Reason:             reasonHealthCheckFail,
	//		LastTransitionTime: now,
	//		LastProbeTime:      now,
	//	})
	//	if err1 := c.persistUpdate(cluster); err1 != nil {
	//		log.Warn("Update cluster status failed", log.String("clusterName", cluster.Name), log.Err(err1))
	//		return err1
	//	}
	//	log.Warn("Failed to build the cluster client", log.String("clusterName", cluster.Name), log.Err(err))
	//	return err
	//}
	//cluster.Status.Resource = *res
	log.Debug("begin to check namespace list", log.String("cluster", cluster.Name))

	_, err = kubeClient.CoreV1().Namespaces().List(metaV1.ListOptions{})
	if err != nil {
		cluster.Status.Phase = v1.ClusterFailed
		cluster.Status.Message = err.Error()
		cluster.Status.Reason = reasonHealthCheckFail
		c.addOrUpdateCondition(cluster, v1.ClusterCondition{
			Type:          conditionTypeHealthCheck,
			Status:        v1.ConditionFalse,
			Message:       err.Error(),
			Reason:        reasonHealthCheckFail,
			LastProbeTime: metaV1.Now(),
		})
	} else {
		cluster.Status.Phase = v1.ClusterRunning
		cluster.Status.Message = ""
		cluster.Status.Reason = ""
		c.addOrUpdateCondition(cluster, v1.ClusterCondition{
			Type:          conditionTypeHealthCheck,
			Status:        v1.ConditionTrue,
			Message:       "",
			Reason:        "",
			LastProbeTime: metaV1.Now(),
		})

		// update version info
		if cluster.Status.Version == "" {
			log.Debug("Update version info", log.String("clusterName", cluster.Name))

		}

		if version, err := kubeClient.ServerVersion(); err == nil {
			entireVersion, err := semver.ParseTolerant(version.GitVersion)
			if err == nil {
				pureVersion := semver.Version{Major: entireVersion.Major, Minor: entireVersion.Minor, Patch: entireVersion.Patch}
				if cluster.Status.Version == "" {
					log.Info("Set cluster version", log.String("clusterName", cluster.Name), log.String("version", pureVersion.String()), log.String("entireVersion", entireVersion.String()))
					cluster.Status.Version = pureVersion.String()
					now := metaV1.Now()
					c.addOrUpdateCondition(cluster, v1.ClusterCondition{
						Type:          conditionTypeSyncVersion,
						Status:        v1.ConditionTrue,
						Message:       "the actual cluster version was detected",
						Reason:        "ClusterVersionSynced",
						LastProbeTime: now,
					})
				} else if cluster.Status.Version != pureVersion.String() {
					log.Info("Update cluster version", log.String("clusterName", cluster.Name), log.String("version", pureVersion.String()), log.String("entireVersion", entireVersion.String()))
					cluster.Status.Version = pureVersion.String()
					now := metaV1.Now()
					c.addOrUpdateCondition(cluster, v1.ClusterCondition{
						Type:          conditionTypeSyncVersion,
						Status:        v1.ConditionTrue,
						Message:       "detected that the cluster version has been upgraded",
						Reason:        "ClusterVersionUpgraded",
						LastProbeTime: now,
					})
				}
			} else {
				log.Error("Failed to check cluster version", log.String("clusterName", cluster.Name), log.String("version", version.GitVersion), log.Err(err))
				c.addOrUpdateCondition(cluster, v1.ClusterCondition{
					Type:          conditionTypeSyncVersion,
					Status:        v1.ConditionFalse,
					Message:       fmt.Sprintf("failed to check cluster version: %v", err),
					Reason:        "ClusterVersionSyncFailure",
					LastProbeTime: metaV1.Now(),
				})
			}
		}
	}
	log.Debug("begin to persistUpdate", log.String("cluster", cluster.Name), log.String("cluster status Phase", string(cluster.Status.Phase)))

	if err1 := c.persistUpdate(cluster); err1 != nil {
		log.Error("Update cluster status failed", log.String("clusterName", cluster.Name), log.Err(err1))
		return err1
	}
	return err
}

// cal the cluster's capacity , allocatable and allocated resource
func (c *Controller) caclClusterResource(kubeClient *kubernetes.Clientset, cluster string) (*v1.ClusterResource, error) {
	// cal the node's capacity and allocatable
	var cpuCapacity, memoryCapcity, cpuAllocatable, memoryAllocatable, cpuAllocated, memoryAllocated resource.Quantity
	var cnt = 0
	var total = 0
	var continueField = ""

	for {
		log.Debug("start to get nodes list, ", log.String("cluster", cluster), log.Int("cnt", cnt))
		nodeList, err := kubeClient.CoreV1().Nodes().List(metaV1.ListOptions{Continue: continueField, Limit: int64(300)})
		if err != nil {
			return &v1.ClusterResource{}, err
		}
		for _, node := range nodeList.Items {
			for resourceName, capacity := range node.Status.Capacity {
				if resourceName.String() == string(coreV1.ResourceCPU) {
					cpuCapacity.Add(capacity)
				}
				if resourceName.String() == string(coreV1.ResourceMemory) {
					memoryCapcity.Add(capacity)
				}
			}

			for resourceName, allocatable := range node.Status.Allocatable {
				if resourceName.String() == string(coreV1.ResourceCPU) {
					cpuAllocatable.Add(allocatable)
				}
				if resourceName.String() == string(coreV1.ResourceMemory) {
					memoryAllocatable.Add(allocatable)
				}
			}
		}
		curSize := len(nodeList.Items)
		total += curSize
		log.Debug("get node size", log.String("cluster", cluster), log.Int("size", curSize), log.Int("total size", total))
		if nodeList.Continue == "" {
			break
		}
		continueField = nodeList.Continue
		log.Debug("next nodeList ", log.String("cluster", cluster), log.String("continue", nodeList.Continue))
		cnt++
	}

	cnt = 0
	total = 0
	continueField = ""

	// cal the pods's request resource as allocated resource
	for {
		log.Debug("start to get pods list, ", log.String("cluster", cluster), log.Int("cnt", cnt))
		podsList, err := kubeClient.CoreV1().Pods("").List(metaV1.ListOptions{Continue: continueField, Limit: int64(500)})
		if err != nil {
			return &v1.ClusterResource{}, err
		}
		for _, pod := range podsList.Items {
			// same with kubectl skip those pods in failed or succeeded status
			if pod.Status.Phase == coreV1.PodFailed || pod.Status.Phase == coreV1.PodSucceeded {
				continue
			}
			for _, container := range pod.Spec.Containers {
				for resourceName, allocated := range container.Resources.Requests {
					if resourceName.String() == string(coreV1.ResourceCPU) {
						cpuAllocated.Add(allocated)
					}
					if resourceName.String() == string(coreV1.ResourceMemory) {
						memoryAllocated.Add(allocated)
					}
				}
			}
		}
		curSize := len(podsList.Items)
		total += curSize
		log.Debug("get pod size", log.String("cluster", cluster), log.Int("size", curSize), log.Int("total", total))

		if podsList.Continue == "" {
			break
		}
		continueField = podsList.Continue
		log.Debug("next podList ", log.String("cluster", cluster), log.String("continue", podsList.Continue))
		cnt++
	}
	result := &v1.ClusterResource{
		Capacity: v1.ResourceList{
			v1.ResourceName(coreV1.ResourceCPU):    cpuCapacity,
			v1.ResourceName(coreV1.ResourceMemory): memoryCapcity,
		},
		Allocatable: v1.ResourceList{
			v1.ResourceName(coreV1.ResourceCPU):    cpuAllocatable,
			v1.ResourceName(coreV1.ResourceMemory): memoryAllocatable,
		},
		Allocated: v1.ResourceList{
			v1.ResourceName(coreV1.ResourceCPU):    cpuAllocated,
			v1.ResourceName(coreV1.ResourceMemory): memoryAllocated,
		},
	}
	return result, nil
}

// for PollImmediateUntil, when return true ,an err while exit
func (c *Controller) watchClusterHealth(clusterName string) func() (bool, error) {
	return func() (bool, error) {
		log.Debug("Check cluster health", log.String("clusterName", clusterName))

		cluster, err := c.client.PlatformV1().Clusters().Get(clusterName, metaV1.GetOptions{})
		if err != nil {
			// if cluster is not found,to exit the health check loop
			if errors.IsNotFound(err) {
				log.Warn("Cluster not found, to exit the health check loop", log.String("clusterName", clusterName))
				return true, nil
			}
			log.Error("Check cluster health, cluster get failed", log.String("clusterName", clusterName), log.Err(err))
			return false, nil
		}

		// if status is terminated,to exit the  health check loop
		if cluster.Status.Phase == v1.ClusterTerminating || cluster.Status.Phase == v1.ClusterInitializing {
			log.Warn("Cluster status is terminated, to exit the health check loop", log.String("clusterName", cluster.Name))
			return true, nil
		}

		_ = c.checkClusterHealth(cluster)
		return false, nil
	}
}
