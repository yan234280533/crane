package nodelocal

import (
	"fmt"
	"time"

	corelisters "k8s.io/client-go/listers/core/v1"

	"github.com/shirou/gopsutil/mem"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/statestore/types"
)

const (
	memoryCollectorName = "memory"
)

func init() {
	registerMetrics(memoryCollectorName, []types.MetricName{types.MetricNameMemoryTotalUsage, types.MetricNameMemoryTotalUtilization}, NewMemoryCollector)
}

type MemoryTimeStampState struct {
	stat      mem.VirtualMemoryStat
	timestamp time.Time
}

type MemoryCollector struct {
	memoryState *MemoryTimeStampState
	data        map[string][]common.TimeSeries
}

// NewMemoryCollector returns a new Collector exposing kernel/system statistics.
func NewMemoryCollector(_ corelisters.PodLister) (nodeLocalCollector, error) {

	klog.V(2).Infof("NewMemoryCollector")

	var data = make(map[string][]common.TimeSeries)

	return &MemoryCollector{data: data}, nil
}

func (c *MemoryCollector) collect() (map[string][]common.TimeSeries, error) {
	var now = time.Now()
	stat, err := mem.VirtualMemory()
	if err != nil {
		return map[string][]common.TimeSeries{}, err
	}

	if stat == nil {
		return map[string][]common.TimeSeries{}, fmt.Errorf("stat is nil")
	}

	nowMemoryState := &MemoryTimeStampState{
		stat:      *stat,
		timestamp: now,
	}

	c.memoryState = nowMemoryState

	usagePercent := c.memoryState.stat.UsedPercent
	usage := c.memoryState.stat.Used

	klog.V(6).Infof("MemoryCollector collected,usagePercent %v, usageCore %v", usagePercent, usage)

	c.data[string(types.MetricNameMemoryTotalUsage)] = []common.TimeSeries{{Samples: []common.Sample{{Value: float64(usage), Timestamp: now.Unix()}}}}
	c.data[string(types.MetricNameCpuTotalUtilization)] = []common.TimeSeries{{Samples: []common.Sample{{Value: usagePercent, Timestamp: now.Unix()}}}}

	return c.data, nil
}

func (c *MemoryCollector) name() string {
	return memoryCollectorName
}
