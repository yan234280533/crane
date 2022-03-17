package evpa

import (
	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
)

const (
	DefaultComponentScaleDownStabWindowSeconds = int32(43200)
	DefaultComponentScaleUpStabWindowSeconds   = int32(150)

	// DefaultScaleDownCPUUtilPercentageThreshold defines the cpu scaledown threshold,,
	// If the ratio of actual used cpu resources divided by request resources is less than DefaultScaleDownCPUUtilPercentageThreshold,
	// it will trigger cpu scaledown.
	DefaultScaleDownCPUUtilPercentageThreshold = int32(35)
	// DefaultScaleUpCPUUtilPercentageThreshold defines the cpu scaleup threshold,,
	// If the ratio of actual used cpu resources divided by limit resources is greater than DefaultScaleUpCPUUtilPercentageThreshold,
	// it will trigger cpu scaleup.
	DefaultScaleUpCPUUtilPercentageThreshold = int32(95)
	// DefaultScaleDownMemoryUtilPercentageThreshold defines the memory scaledown threshold,,
	// If the ratio of actual used memory resources divided by request resources is less than DefaultScaleDownMemoryUtilPercentageThreshold,
	// it will trigger memory scaledown.
	DefaultScaleDownMemoryUtilPercentageThreshold = int32(40)
	// DefaultScaleUpMemoryUtilPercentageThreshold defines the memory scaleup threshold,,
	// If the ratio of actual used cpu resources divided by limit resources is greater than DefaultScaleUpCPUUtilPercentageThreshold,
	// it will trigger memory scaleup.
	DefaultScaleUpMemoryUtilPercentageThreshold = int32(95)
)

var (
	DefaultControlledResources = []autoscalingapi.ResourceName{autoscalingapi.ResourceName("cpu"), autoscalingapi.ResourceName("memory")}

	defaultEstimators = []autoscalingapi.ResourceEstimator{
		{
			Type:   "Percentile",
			Config: map[string]string{},
		},
		{
			Type:   "Proportional",
			Config: map[string]string{},
		},
		{
			Type:   "OOM",
			Config: map[string]string{},
		},
	}
)
