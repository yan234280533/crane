package nodelocal

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gocrane/crane/pkg/common"
	cmemory "github.com/google/cadvisor/cache/memory"
	cadvisorcontainer "github.com/google/cadvisor/container"
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"
	cmanager "github.com/google/cadvisor/manager"
	csysfs "github.com/google/cadvisor/utils/sysfs"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/ensurance/statestore/types"
	"github.com/gocrane/crane/pkg/log"
)

const (
	cadvisorCollectorName = "cadvisor"
)

func init() {
	registerMetrics(cadvisorCollectorName, []types.MetricName{types.MetricNameContainerCpuTotalUsage, types.MetricNameContainerSchedRunQueueTime}, NewCadvisor)
}

type CgroupState struct {
	stat      cadvisorapiv2.ContainerInfo
	timestamp time.Time
}

//CadvisorCollector is the collector to collect container state
type CadvisorCollector struct {
	Manager   cmanager.Manager
	podLister corelisters.PodLister

	cgroupState           map[string]CgroupState
	MemCache              *cmemory.InMemoryCache
	SysFs                 csysfs.SysFs
	IncludeMetrics        cadvisorcontainer.MetricSet
	MaxHousekeepingConfig cmanager.HouskeepingConfig
}

func NewCadvisorManager(podLister corelisters.PodLister) (*CadvisorCollector, error) {
	log.Logger().V(1).Info("NewCadvisorManager")

	var includedMetrics = cadvisorcontainer.MetricSet{
		cadvisorcontainer.CpuUsageMetrics:         struct{}{},
		cadvisorcontainer.ProcessSchedulerMetrics: struct{}{},
	}

	var allowDynamic bool = true
	var maxHousekeepingInterval time.Duration = 10 * time.Second
	var memCache = cmemory.New(10*time.Minute, nil)
	var sysfs = csysfs.NewRealSysFs()
	var maxHousekeepingConfig = cmanager.HouskeepingConfig{Interval: &maxHousekeepingInterval, AllowDynamic: &allowDynamic}

	m, err := cmanager.New(memCache, sysfs, maxHousekeepingConfig, includedMetrics, http.DefaultClient, []string{"/kubepods"}, "")
	if err != nil {
		return nil, fmt.Errorf("cadvisor manager start err: %s", err.Error())
	}

	c := CadvisorCollector{
		Manager:     m,
		podLister:   podLister,
		cgroupState: make(map[string]CgroupState, 0),
	}

	c.Manager.Start()

	return &c, nil
}

func NewCadvisor(podLister corelisters.PodLister) (nodeLocalCollector, error) {
	log.Logger().V(1).Info("NewCadvisor")

	var includedMetrics = cadvisorcontainer.MetricSet{
		cadvisorcontainer.CpuUsageMetrics:         struct{}{},
		cadvisorcontainer.ProcessSchedulerMetrics: struct{}{},
	}

	var allowDynamic bool = true
	var maxHousekeepingInterval time.Duration = 10 * time.Second
	var memCache = cmemory.New(10*time.Minute, nil)
	var sysfs = csysfs.NewRealSysFs()
	var maxHousekeepingConfig = cmanager.HouskeepingConfig{Interval: &maxHousekeepingInterval, AllowDynamic: &allowDynamic}

	m, err := cmanager.New(memCache, sysfs, maxHousekeepingConfig, includedMetrics, http.DefaultClient, []string{types.CgroupKubePods}, "")
	if err != nil {
		return nil, fmt.Errorf("cadvisor manager start err: %s", err.Error())
	}

	c := CadvisorCollector{
		Manager:   m,
		podLister: podLister,
	}

	c.Manager.Start()

	return &c, nil
}

// Start cadvisor manager
func (c *CadvisorCollector) Start() error {
	return c.Manager.Start()
}

// Stop cadvisor and clear existing factory
func (c *CadvisorCollector) Stop() error {
	if err := c.Manager.Stop(); err != nil {
		return err
	}

	// clear existing factory
	cadvisorcontainer.ClearContainerHandlerFactories()

	return nil
}

func (c *CadvisorCollector) name() string {
	return cadvisorCollectorName
}

func (c *CadvisorCollector) collect() (map[string][]common.TimeSeries, error) {
	var cgroupState = make(map[string]CgroupState, 0)

	allPods, err := c.podLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list all pods: %v", err)
		return make(map[string][]common.TimeSeries, 0), err
	}

	var cpuUsageTimeSeries []common.TimeSeries
	var schedRunQueueTimeSeries []common.TimeSeries
	var cpuLimitTimeSeries []common.TimeSeries
	var cpuQuotaTimeSeries []common.TimeSeries
	var cpuPeriodTimeSeries []common.TimeSeries

	for _, pod := range allPods {
		var ref = GetCgroupRefFromPod(pod)
		var now = time.Now()

		containerInfo, err := c.Manager.GetContainerInfoV2(ref.GetCgroupPath(), cadvisorapiv2.RequestOptions{
			IdType:    cadvisorapiv2.TypeName,
			Count:     1,
			Recursive: true,
		})

		if err != nil {
			klog.Errorf("GetContainerInfoV2 failed: %v", err)
			continue
		}

		for key, v := range containerInfo {
			var containerId = GetContainerIdFromKey(key)
			var containerName = GetContainerNameFromPod(pod, containerId)
			var refCopy = ref
			refCopy.ContainerId = containerId
			refCopy.ContainerName = containerName

			if state, ok := c.cgroupState[key]; ok {
				var label = GetLabelFromRef(&ref)
				cpuUsageIncrease := v.Stats[0].Cpu.Usage.Total - state.stat.Stats[0].Cpu.Usage.Total
				schedRunqueueTimeIncrease := v.Stats[0].Cpu.Schedstat.RunqueueTime - state.stat.Stats[0].Cpu.Schedstat.RunqueueTime
				timeIncrease := v.Stats[0].Timestamp.UnixNano() - state.stat.Stats[0].Timestamp.UnixNano()
				cpuUsage := float64(cpuUsageIncrease) / float64(timeIncrease)
				schedRunqueueTime := float64(schedRunqueueTimeIncrease) * 1000 * 1000 / float64(timeIncrease)

				cpuUsageTimeSeries = append(cpuUsageTimeSeries, common.TimeSeries{Labels: label, Samples: []common.Sample{{Value: cpuUsage, Timestamp: now.Unix()}}})
				schedRunQueueTimeSeries = append(schedRunQueueTimeSeries, common.TimeSeries{Labels: label, Samples: []common.Sample{{Value: schedRunqueueTime, Timestamp: now.Unix()}}})
				cpuLimitTimeSeries = append(cpuLimitTimeSeries, common.TimeSeries{Labels: label, Samples: []common.Sample{{Value: float64(state.stat.Spec.Cpu.Limit), Timestamp: now.Unix()}}})
				cpuQuotaTimeSeries = append(cpuQuotaTimeSeries, common.TimeSeries{Labels: label, Samples: []common.Sample{{Value: float64(state.stat.Spec.Cpu.Quota), Timestamp: now.Unix()}}})
				cpuPeriodTimeSeries = append(cpuPeriodTimeSeries, common.TimeSeries{Labels: label, Samples: []common.Sample{{Value: float64(state.stat.Spec.Cpu.Period), Timestamp: now.Unix()}}})
			}

			cgroupState[key] = CgroupState{stat: v, timestamp: now}
		}
	}

	c.cgroupState = cgroupState

	var storeMaps = make(map[string][]common.TimeSeries, 0)
	storeMaps[string(types.MetricNameContainerCpuTotalUsage)] = cpuUsageTimeSeries
	storeMaps[string(types.MetricNameContainerSchedRunQueueTime)] = schedRunQueueTimeSeries
	storeMaps[string(types.MetricNameContainerCpuLimit)] = cpuLimitTimeSeries
	storeMaps[string(types.MetricNameContainerCpuQuota)] = cpuQuotaTimeSeries
	storeMaps[string(types.MetricNameContainerCpuPeriod)] = cpuPeriodTimeSeries

	return storeMaps, nil
}

func GetCgroupRefFromPod(pod *v1.Pod) types.CgroupRef {
	var ref types.CgroupRef

	ref.PodQOSClass = pod.Status.QOSClass
	ref.PodName = pod.Name
	ref.PodNamespace = pod.Namespace
	ref.PodUid = string(pod.UID)

	return ref
}

func GetContainerIdFromKey(key string) string {
	subPaths := strings.Split(key, "/")

	if len(subPaths) > 0 {
		// if the latest sub path is pod-xxx-xxx, we regard as it od path
		// if not we used the latest sub path as the containerId
		if strings.HasPrefix(subPaths[len(subPaths)-1], types.CgroupPodPrefix) {
			return ""
		} else {
			return subPaths[len(subPaths)-1]
		}
	}

	return ""
}

func GetContainerNameFromPod(pod *v1.Pod, containerId string) string {
	if containerId == "" {
		return ""
	}

	for _, v := range pod.Status.ContainerStatuses {
		strList := strings.Split(v.ContainerID, "//")
		if len(strList) > 0 {
			if strList[len(strList)-1] == containerId {
				return v.Name
			}
		}
	}

	return ""
}

func GetLabelFromRef(ref *types.CgroupRef) []common.Label {
	return []common.Label{
		{Name: "PodName", Value: ref.PodName},
		{Name: "PodNamespace", Value: ref.PodNamespace},
		{Name: "PodUid", Value: ref.PodUid},
		{Name: "ContainerName", Value: ref.ContainerName},
		{Name: "ContainerId", Value: ref.ContainerId},
	}
}
