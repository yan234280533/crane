package analyzer

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type AvoidanceActionStruct struct {
	BlockScheduledAction *BlockScheduledActionStruct
	ThrottleActions      []ThrottleActionStruct
	EvictActions         []EvictActionStruct
}

type BlockScheduledActionStruct struct {
	BlockScheduledQOSPriority   *ScheduledQOSPriority
	RestoreScheduledQOSPriority *ScheduledQOSPriority
}

type ScheduledQOSPriority struct {
	PodQOSClass        v1.PodQOSClass
	PriorityClassValue uint64
}

type CPUThrottleActionStruct struct {
	CPUDownAction *CPURatioStruct
	CPUUpAction   *CPURatioStruct
}

type CPURatioStruct struct {
	//the min of cpu ratio for pods
	// +optional
	MinCPURatio uint64 `json:"minCPURatio,omitempty"`

	//the step of cpu share and limit for once down-size (1-100)
	// +optional
	StepCPURatio uint64 `json:"stepCPURatio,omitempty"`
}

type MemoryThrottleActionStruct struct {
	// to force gc the page cache of low level pods
	// +optional
	ForceGC bool `json:"forceGC,omitempty"`
}

type ThrottleActionStruct struct {
	CPUThrottle    *CPUThrottleActionStruct
	MemoryThrottle *MemoryThrottleActionStruct
	ThrottlePods   []types.NamespacedName
}

type EvictActionStruct struct {
	DeletionGracePeriodSeconds *int32 `json:"deletionGracePeriodSeconds,omitempty"`
	EvictPods                  []types.NamespacedName
}
