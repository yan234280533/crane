package avoidance

import (
	"fmt"
	"github.com/gocrane-io/crane/pkg/ensurance/analyzer"

	"sync"
	"time"

	"github.com/gocrane-io/crane/pkg/utils/clogs"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	ecache "github.com/gocrane-io/crane/pkg/ensurance/cache"
	einformer "github.com/gocrane-io/crane/pkg/ensurance/informer"
)

type AvoidanceManager struct {
	nodeName                string
	client                  clientset.Interface
	podInformer             cache.SharedIndexInformer
	nodeInformer            cache.SharedIndexInformer
	avoidanceInformer       cache.SharedIndexInformer
	detectionConditionCache ecache.DetectionConditionCache
	dcsOlder                []ecache.DetectionCondition
	NodeStatus              sync.Map
}

// AvoidanceManager create avoidance manager
func NewAvoidanceManager(client clientset.Interface, nodeName string, podInformer cache.SharedIndexInformer, nodeInformer cache.SharedIndexInformer, avoidanceInformer cache.SharedIndexInformer,
	detectionConditionCache ecache.DetectionConditionCache) *AvoidanceManager {
	return &AvoidanceManager{
		nodeName:                nodeName,
		client:                  client,
		podInformer:             podInformer,
		nodeInformer:            nodeInformer,
		avoidanceInformer:       avoidanceInformer,
		detectionConditionCache: detectionConditionCache,
	}
}

// Run does nothing
func (a *AvoidanceManager) Run(stop <-chan struct{}) {
	clogs.Log().Info("Avoidance manager starts running")

	go func() {
		updateTicker := time.NewTicker(10 * time.Second)
		defer updateTicker.Stop()

		for {
			select {
			case <-updateTicker.C:
				clogs.Log().Info("Avoidance run periodically")
				a.runOnce(stop)
			case <-stop:
				{
					clogs.Log().Info("Avoidance stop event")
					return
				}
				/*case event := <-a.eventChan:
				clogs.Log().Info("receive avoidance event: %+v", event)
				a.runOnce()
				*/
			}
		}
	}()

	return
}

func (a *AvoidanceManager) runOnce(stop <-chan struct{}) error {
	//step 1: get detection state
	dcs := a.detectionConditionCache.ListDetections()

	//step 2: print log and event
	a.doLogEvent(dcs)

	//step 3: merge detection state
	avoidanceActionStruct, err := a.doMerge(dcs, stop)
	if err != nil {
		return err
	}

	//step 4: merge detection state
	if err = a.doAction(avoidanceActionStruct, stop); err != nil {
		return err
	}

	clogs.Log().V(5).Info("AvoidanceManager runOnce succeed")
	return nil
}

func (a *AvoidanceManager) doLogEvent(dcs []ecache.DetectionCondition) {
	//step1 print log if the detection state is changed
	//step2 produce event
}

func (a *AvoidanceManager) doMerge(dcs []ecache.DetectionCondition, stop <-chan struct{}) (analyzer.AvoidanceActionStruct, error) {
	//step1 filter the only dryRun detection
	//step2 do BlockScheduled merge
	//step3 do Throttle merge FilterAndSortThrottlePods
	//step3 do Evict merge  FilterAndSortEvictPods
	return analyzer.AvoidanceActionStruct{}, nil
}

func (a *AvoidanceManager) doAction(s analyzer.AvoidanceActionStruct, stop <-chan struct{}) error {
	//step1 do BlockScheduled action
	if err := a.blockScheduled(s.BlockScheduledAction, stop); err != nil {
		return err
	}

	//step2 do Evict action
	if err := a.evictAction(s.EvictActions, stop); err != nil {
		return err
	}

	//step3 do Throttle action

	return nil
}

func (a *AvoidanceManager) blockScheduled(bsa *analyzer.BlockScheduledActionStruct, stop <-chan struct{}) error {
	// step1: get node
	node, err := einformer.GetNodeFromInformer(a.nodeInformer, a.nodeName)
	if err != nil {
		return err
	}

	clogs.Log().V(6).Info(fmt.Sprintf("node condition %+v", node.Status.Conditions))

	// step2 update node condition for block scheduled
	if bsa.BlockScheduledQOSPriority != nil {
		// einformer.updateNodeConditions
		// einformer.updateNodeStatus
	}

	// step2 update node condition for restored scheduled
	if bsa.RestoreScheduledQOSPriority != nil {
		// einformer.updateNodeConditions
		// einformer.updateNodeStatus
	}

	return nil
}

func (a *AvoidanceManager) evictAction(ea []analyzer.EvictActionStruct, stop <-chan struct{}) error {
	var bSucceed bool

	for _, e := range ea {
		for _, podNamespace := range e.EvictPods {
			pod, err := einformer.GetPodFromInformer(a.podInformer, podNamespace.String())
			if err != nil {
				bSucceed = false
				continue
			}
			clogs.Log().V(5).Info("pod %+v", pod)
			//go einformer.EvictPodWithGracePeriod(a.client,pod,einformer.GetGracePeriodSeconds(e.DeletionGracePeriodSeconds))
		}
	}

	if !bSucceed {
		return fmt.Errorf("some pod evict failed")
	}

	return nil
}

func (a *AvoidanceManager) throttleAction(bsa *analyzer.BlockScheduledActionStruct, stop <-chan struct{}) error {

}
