package nodelocal

import (
	"fmt"
	"math"
	"time"

	"github.com/gocrane-io/crane/pkg/ensurance/statestore/types"

	"github.com/shirou/gopsutil/cpu"

	"github.com/gocrane-io/crane/pkg/utils"
)

const (
	cpuCollectorName = "cpu"
)

func init() {
	registerMetrics(cpuCollectorName, []types.MetricName{types.MetricNamCpuTotalUsage, types.MetricNamCpuTotalUtilization}, NewCPUCollector)
}

type CpuTimeStampState struct {
	stat      cpu.TimesStat
	timestamp time.Time
}

type CpuCollector struct {
	cpuState       *CpuTimeStampState
	cpuCoreNumbers uint64
	data           map[string]utils.TimeSeries
}

// NewCPUCollector returns a new Collector exposing kernel/system statistics.
func NewCPUCollector() (nodeLocalCollector, error) {
	var cpuCoreNumbers uint64
	if cpuInfos, err := cpu.Info(); err != nil {
		return nil, err
	} else {
		cpuCoreNumbers = uint64(len(cpuInfos))
	}

	return &CpuCollector{cpuCoreNumbers: cpuCoreNumbers}, nil
}

func (c *CpuCollector) collect() (map[string]utils.TimeSeries, error) {
	var now = time.Now()
	stats, err := cpu.Times(false)
	if err != nil {
		return map[string]utils.TimeSeries{}, err
	}

	if len(stats) != 1 {
		return map[string]utils.TimeSeries{}, fmt.Errorf("len stat is not 1")
	}

	nowCpuState := &CpuTimeStampState{
		stat:      stats[0],
		timestamp: now,
	}

	if c.cpuState == nil {
		c.cpuState = nowCpuState
		return map[string]utils.TimeSeries{}, fmt.Errorf("collect_init")
	}

	usagePercent := calculateBusy(c.cpuState.stat, nowCpuState.stat)
	usageCore := usagePercent * float64(c.cpuCoreNumbers)

	c.cpuState = nowCpuState

	c.data[string(types.MetricNamCpuTotalUsage)] = utils.TimeSeries{Samples: []utils.Sample{{Value: usageCore, Timestamp: now.Unix()}}}
	c.data[string(types.MetricNamCpuTotalUtilization)] = utils.TimeSeries{Samples: []utils.Sample{{Value: usagePercent, Timestamp: now.Unix()}}}
	return c.data, nil
}

func (c *CpuCollector) name() string {
	return cpuCollectorName
}

func calculateBusy(stat1 cpu.TimesStat, stat2 cpu.TimesStat) float64 {
	stat1All, stat1Busy := getAllBusy(stat1)
	stat2All, stat2Busy := getAllBusy(stat1)

	if stat2Busy <= stat1Busy {
		return 0
	}
	if stat2All <= stat1All {
		return 100
	}
	return math.Min(100, math.Max(0, (stat2Busy-stat1Busy)/(stat2All-stat1All)*100))
}

func getAllBusy(stat cpu.TimesStat) (float64, float64) {
	busy := stat.User + stat.System + stat.Nice + stat.Iowait + stat.Irq + stat.Softirq + stat.Steal
	return busy + stat.Idle, busy
}
