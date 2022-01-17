package nodelocal

import (
	"io/ioutil"
	"time"

	"github.com/shirou/gopsutil/disk"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/statestore/types"
)

const (
	diskioCollectorName = "diskio"
	sysBlockPath        = "/sys/block"
)

func init() {
	registerMetrics(diskioCollectorName, []types.MetricName{types.MetricDiskReadKiBPS, types.MetricDiskWriteKiBPS, types.MetricDiskReadIOPS, types.MetricDiskWriteIOPS, types.MetricDiskUtilization}, NewDiskIOCollector)
}

type DiskTimeStampState struct {
	stat      disk.IOCountersStat
	timestamp time.Time
}

type DiskIOCollector struct {
	diskStates map[string]DiskTimeStampState
}

type DiskIOUsage struct {
	DiskReadKiBps  float64
	DiskWriteKiBps float64
	DiskReadIOps   float64
	DiskWriteIOps  float64
	Utilization    float64
}

// NewDiskIOCollector returns a new Collector exposing kernel/system statistics.
func NewDiskIOCollector(_ corelisters.PodLister) (nodeLocalCollector, error) {

	klog.V(2).Infof("NewDiskIOCollector")

	return &DiskIOCollector{diskStates: make(map[string]DiskTimeStampState)}, nil
}

func (d *DiskIOCollector) collect() (map[string][]common.TimeSeries, error) {
	var now = time.Now()

	devices, err := SysBlockDevices(sysBlockPath)
	if err != nil {
		return map[string][]common.TimeSeries{}, err
	}

	diskIOStats, err := disk.IOCounters(devices...)
	if err != nil {
		klog.Errorf("Failed to collect disk io resource: %v", err)
		return map[string][]common.TimeSeries{}, err
	}

	var diskReadKiBpsTimeSeries []common.TimeSeries
	var diskWriteKiBpsTimeSeries []common.TimeSeries
	var diskReadIOpsTimeSeries []common.TimeSeries
	var diskWriteIOpsTimeSeries []common.TimeSeries
	var diskUtilizationTimeSeries []common.TimeSeries

	var diskStateMaps = make(map[string]DiskTimeStampState)
	for key, v := range diskIOStats {
		diskStateMaps[key] = DiskTimeStampState{stat: v, timestamp: now}
		if vv, ok := d.diskStates[key]; ok {
			diskIOUsage := calculateDiskIO(diskStateMaps[key], vv)
			diskReadKiBpsTimeSeries = append(diskReadKiBpsTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "diskName", Value: key}}, Samples: []common.Sample{{Value: diskIOUsage.DiskReadKiBps, Timestamp: now.Unix()}}})
			diskWriteKiBpsTimeSeries = append(diskWriteKiBpsTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "diskName", Value: key}}, Samples: []common.Sample{{Value: diskIOUsage.DiskWriteKiBps, Timestamp: now.Unix()}}})
			diskReadIOpsTimeSeries = append(diskReadIOpsTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "diskName", Value: key}}, Samples: []common.Sample{{Value: diskIOUsage.DiskReadIOps, Timestamp: now.Unix()}}})
			diskWriteIOpsTimeSeries = append(diskWriteIOpsTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "diskName", Value: key}}, Samples: []common.Sample{{Value: diskIOUsage.DiskWriteIOps, Timestamp: now.Unix()}}})
			diskUtilizationTimeSeries = append(diskUtilizationTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "diskName", Value: key}}, Samples: []common.Sample{{Value: diskIOUsage.Utilization, Timestamp: now.Unix()}}})
		}
	}

	d.diskStates = diskStateMaps

	var storeMaps = make(map[string][]common.TimeSeries, 0)
	storeMaps[string(types.MetricDiskReadKiBPS)] = diskReadKiBpsTimeSeries
	storeMaps[string(types.MetricDiskWriteKiBPS)] = diskWriteKiBpsTimeSeries
	storeMaps[string(types.MetricDiskReadIOPS)] = diskReadIOpsTimeSeries
	storeMaps[string(types.MetricDiskWriteIOPS)] = diskWriteIOpsTimeSeries
	storeMaps[string(types.MetricDiskUtilization)] = diskUtilizationTimeSeries

	return storeMaps, nil
}

func (d *DiskIOCollector) name() string {
	return diskioCollectorName
}

// SysBlockDevices lists the device names from /sys/block/<dev>.
// Copied from https://github.com/prometheus/procfs/blob/master/blockdevice/stats.go
func SysBlockDevices(sysBlockPath string) ([]string, error) {
	deviceDirs, err := ioutil.ReadDir(sysBlockPath)
	if err != nil {
		return nil, err
	}
	devices := []string{}
	for _, deviceDir := range deviceDirs {
		devices = append(devices, deviceDir.Name())
	}
	return devices, nil
}

// calculateDiskIO calculate disk io usage
func calculateDiskIO(stat1 DiskTimeStampState, stat2 DiskTimeStampState) DiskIOUsage {

	duration := float64(stat2.timestamp.Unix() - stat1.timestamp.Unix())

	return DiskIOUsage{
		DiskReadKiBps:  float64(stat2.stat.ReadBytes-stat1.stat.ReadBytes) / 1024.0 / duration,
		DiskWriteKiBps: float64(stat2.stat.WriteBytes-stat1.stat.WriteBytes) / 1024.0 / duration,
		DiskReadIOps:   float64(stat2.stat.ReadCount-stat1.stat.ReadCount) / duration,
		DiskWriteIOps:  float64(stat2.stat.WriteCount-stat1.stat.WriteCount) / duration,
		Utilization:    (float64(stat2.stat.IoTime-stat1.stat.IoTime) / 1000.0 / duration) * 100,
	}
}
