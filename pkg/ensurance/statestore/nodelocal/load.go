package nodelocal

import (
	"fmt"
	"github.com/gocrane/crane/pkg/common"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
	"time"

	"github.com/shirou/gopsutil/load"

	"github.com/gocrane/crane/pkg/ensurance/statestore/types"
)

const (
	loadCollectorName = "load"
)

func init() {
	registerMetrics(loadCollectorName, []types.MetricName{types.MetricNameCpuLoad1Min, types.MetricNameCpuLoad5Min, types.MetricNameCpuLoad15Min}, NewLoadCollector)
}

type LoadTimeStampState struct {
	stat      load.AvgStat
	timestamp time.Time
}

type LoadCollector struct {
	loadState *LoadTimeStampState
	data      map[string][]common.TimeSeries
}

// NewLoadCollector returns a new Collector exposing kernel/system statistics.
func NewLoadCollector(_ corelisters.PodLister) (nodeLocalCollector, error) {

	klog.V(2).Infof("NewLoadCollector")

	var data = make(map[string][]common.TimeSeries)

	return &LoadCollector{data: data}, nil
}

func (l *LoadCollector) collect() (map[string][]common.TimeSeries, error) {
	var now = time.Now()
	stat, err := load.Avg()
	if err != nil {
		return map[string][]common.TimeSeries{}, err
	}

	if stat == nil {
		return map[string][]common.TimeSeries{}, fmt.Errorf("stat is nil")
	}

	l.loadState = &LoadTimeStampState{
		stat:      *stat,
		timestamp: now,
	}

	klog.V(6).Infof("LoadCollector collected,1minLoad %v, 5minLoad %v, 15minLoad", stat.Load1, stat.Load5, stat.Load15)

	l.data[string(types.MetricNameCpuLoad1Min)] = []common.TimeSeries{{Samples: []common.Sample{{Value: stat.Load1, Timestamp: now.Unix()}}}}
	l.data[string(types.MetricNameCpuLoad5Min)] = []common.TimeSeries{{Samples: []common.Sample{{Value: stat.Load5, Timestamp: now.Unix()}}}}
	l.data[string(types.MetricNameCpuLoad15Min)] = []common.TimeSeries{{Samples: []common.Sample{{Value: stat.Load15, Timestamp: now.Unix()}}}}

	return l.data, nil
}

func (l *LoadCollector) name() string {
	return loadCollectorName
}
