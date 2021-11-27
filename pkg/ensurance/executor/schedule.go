package executor

import (
	v1 "k8s.io/api/core/v1"

	einformer "github.com/gocrane-io/crane/pkg/ensurance/informer"
	"github.com/gocrane-io/crane/pkg/utils/clogs"
)

type BlockScheduledExecutor struct {
	BlockScheduledQOSPriority   *ScheduledQOSPriority
	RestoreScheduledQOSPriority *ScheduledQOSPriority
}

type ScheduledQOSPriority struct {
	PodQOSClass        v1.PodQOSClass
	PriorityClassValue uint64
}

func (b *BlockScheduledExecutor) Avoid(ctx *ExecuteContext) error {
	clogs.Log().V(6).Info("Avoid", *b)

	if b.BlockScheduledQOSPriority == nil {
		return nil
	}

	node, err := einformer.GetNodeFromInformer(ctx.NodeInformer, ctx.NodeName)
	if err != nil {
		return err
	}

	// update node condition for block scheduled
	if updateNode, needUpdate := einformer.UpdateNodeConditions(node, v1.NodeCondition{Type: einformer.NodeUnscheduledLow, Status: v1.ConditionTrue}); needUpdate {
		if err := einformer.UpdateNodeStatus(ctx.Client, updateNode, nil); err != nil {
			return err
		}
	}

	return nil
}

func (b *BlockScheduledExecutor) Restore(ctx *ExecuteContext) error {
	clogs.Log().V(6).Info("Restore", *b)

	if b.RestoreScheduledQOSPriority == nil {
		return nil
	}

	node, err := einformer.GetNodeFromInformer(ctx.NodeInformer, ctx.NodeName)
	if err != nil {
		return err
	}

	// update node condition for restored scheduled
	if updateNode, needUpdate := einformer.UpdateNodeConditions(node, v1.NodeCondition{Type: einformer.NodeUnscheduledLow, Status: v1.ConditionFalse}); needUpdate {
		if err := einformer.UpdateNodeStatus(ctx.Client, updateNode, nil); err != nil {
			return err
		}
	}

	return nil
}
