package types

import (
	"strings"

	v1 "k8s.io/api/core/v1"

	"github.com/gocrane/crane/pkg/utils"
)

type CollectType string

type MetricName string

const (
	MetricNameCpuTotalUsage       MetricName = "cpu_total_usage"
	MetricNameCpuTotalUtilization MetricName = "cpu_total_utilization"
	MetricNameCpuLoad1Min         MetricName = "cpu_load_1_min"
	MetricNameCpuLoad5Min         MetricName = "cpu_load_5_min"
	MetricNameCpuLoad15Min        MetricName = "cpu_load_15_min"

	MetricNameMemoryTotalUsage       MetricName = "memory_total_usage"
	MetricNameMemoryTotalUtilization MetricName = "memory_total_utilization"

	MetricDiskReadKiBPS   MetricName = "disk_read_kibps"
	MetricDiskWriteKiBPS  MetricName = "disk_write_kibps"
	MetricDiskReadIOPS    MetricName = "disk_read_iops"
	MetricDiskWriteIOPS   MetricName = "disk_read_iops"
	MetricDiskUtilization MetricName = "disk_read_utilization"

	MetricNetworkReceiveKiBPS MetricName = "network_receive_kibps"
	MetricNetworkSentKiBPS    MetricName = "network_sent_kibps"
	MetricNetworkReceivePckPS MetricName = "network_receive_pckps"
	MetricNetworkSentPckPS    MetricName = "network_sent_pckps"
	MetricNetworkDropIn       MetricName = "network_drop_in"
	MetricNetworkDropOut      MetricName = "network_drop_out"

	MetricNameContainerCpuTotalUsage     MetricName = "container_cpu_total_usage"
	MetricNameContainerCpuLimit          MetricName = "container_cpu_limit"
	MetricNameContainerCpuQuota          MetricName = "container_cpu_quota"
	MetricNameContainerCpuPeriod         MetricName = "container_cpu_period"
	MetricNameContainerSchedRunQueueTime MetricName = "container_sched_run_queue_time"
)

const (
	NodeLocalCollectorType CollectType = "node-local"
)

type MetricNameConfig struct {
	//metricName string
	//selector   metav1.LabelSelector
}

type MetricNameConfigs []MetricNameConfig

type UpdateEvent struct {
	Index uint64
}

// CgroupRef group pod infos
type CgroupRef struct {
	ContainerName string
	ContainerId   string
	PodName       string
	PodNamespace  string
	PodUid        string
	PodQOSClass   v1.PodQOSClass
}

func (c *CgroupRef) GetCgroupPath() string {
	var pathArrays = []string{utils.CgroupKubePods}

	switch c.PodQOSClass {
	case v1.PodQOSGuaranteed:
		pathArrays = append(pathArrays, utils.CgroupPodPrefix+c.PodUid)
	case v1.PodQOSBurstable:
		pathArrays = append(pathArrays, strings.ToLower(string(v1.PodQOSBurstable)), utils.CgroupPodPrefix+c.PodUid)
	case v1.PodQOSBestEffort:
		pathArrays = append(pathArrays, strings.ToLower(string(v1.PodQOSBestEffort)), utils.CgroupPodPrefix+c.PodUid)
	default:
		return ""
	}

	if c.ContainerId != "" {
		pathArrays = append(pathArrays, c.ContainerId)
	}

	return strings.Join(pathArrays, "/")
}
