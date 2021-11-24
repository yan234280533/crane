package avoidance

import (
	"fmt"
	"github.com/gocrane-io/crane/pkg/ensurance/analyzer"

	"github.com/gocrane-io/crane/pkg/utils/clogs"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	einformer "github.com/gocrane-io/crane/pkg/ensurance/informer"
)

type AvoidanceManager struct {
	nodeName          string
	client            clientset.Interface
	noticeCh          <-chan analyzer.AvoidanceActionStruct
	podInformer       cache.SharedIndexInformer
	nodeInformer      cache.SharedIndexInformer
	avoidanceInformer cache.SharedIndexInformer
}

// AvoidanceManager create avoidance manager
func NewAvoidanceManager(client clientset.Interface, nodeName string, podInformer cache.SharedIndexInformer, nodeInformer cache.SharedIndexInformer,
	avoidanceInformer cache.SharedIndexInformer, noticeCh <-chan analyzer.AvoidanceActionStruct) *AvoidanceManager {
	return &AvoidanceManager{
		nodeName:          nodeName,
		client:            client,
		noticeCh:          noticeCh,
		podInformer:       podInformer,
		nodeInformer:      nodeInformer,
		avoidanceInformer: avoidanceInformer,
	}
}

// Run does nothing
func (a *AvoidanceManager) Run(stop <-chan struct{}) {
	clogs.Log().Info("Avoidance manager starts running")

	go func() {
		for {
			select {
			case as := <-a.noticeCh:
				clogs.Log().V(5).Info("Avoidance by analyzer noticed")
				if err := a.doAction(as, stop); err != nil {
					// TODO: if it failed in action, how to retry
				}
			case <-stop:
				{
					clogs.Log().Info("Avoidance exit")
					return
				}
			}
		}
	}()

	return
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
	if err := a.throttleAction(s.ThrottleActions, stop); err != nil {
		return err
	}

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

func (a *AvoidanceManager) throttleAction(tas []analyzer.ThrottleActionStruct, stop <-chan struct{}) error {
	return nil
}
