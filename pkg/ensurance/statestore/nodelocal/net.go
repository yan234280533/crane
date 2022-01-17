package nodelocal

import (
	"time"

	"github.com/shirou/gopsutil/net"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/statestore/types"
)

const (
	netioCollectorName = "netio"
)

func init() {
	registerMetrics(netioCollectorName, []types.MetricName{types.MetricNetworkReceiveKiBPS, types.MetricNetworkSentKiBPS, types.MetricNetworkReceivePckPS, types.MetricNetworkSentPckPS, types.MetricNetworkDropIn, types.MetricNetworkDropOut}, NewNetIOCollector)
}

type NetTimeStampState struct {
	stat      net.IOCountersStat
	timestamp time.Time
}

// NetInterfaceUsage records the network usage
type NetInterfaceUsage struct {
	// ReceiveKibps is the kilobits per second for ingress
	ReceiveKibps float64
	// SentKibps is the kilobits per second for egress
	SentKibps float64
	// ReceivePckps is the package per second for ingress
	ReceivePckps float64
	// SentPckps is the package per second for egress
	SentPckps float64
	// DropIn is the package dropped per second for ingress
	DropIn float64
	// DropOut is the package dropped per second for egress
	DropOut float64
}

type NetIOCollector struct {
	netStates map[string]NetTimeStampState
}

// NewNetIOCollector returns a new Collector exposing kernel/system statistics.
func NewNetIOCollector(_ corelisters.PodLister) (nodeLocalCollector, error) {

	klog.V(2).Infof("NewNetIOCollector")

	return &NetIOCollector{netStates: make(map[string]NetTimeStampState)}, nil
}

func (n *NetIOCollector) collect() (map[string][]common.TimeSeries, error) {
	var now = time.Now()

	netIOStats, err := net.IOCounters(true)
	if err != nil {
		klog.Errorf("Failed to collect net io resource: %v", err)
		return map[string][]common.TimeSeries{}, err
	}

	var netReceiveKiBpsTimeSeries []common.TimeSeries
	var netSentKiBpsTimeSeries []common.TimeSeries
	var netReceivePckpsTimeSeries []common.TimeSeries
	var netSentPckpsTimeSeries []common.TimeSeries
	var netDropInTimeSeries []common.TimeSeries
	var netDropOutTimeSeries []common.TimeSeries

	var netStateMaps = make(map[string]NetTimeStampState)
	for _, v := range netIOStats {
		if v.Name == "" {
			continue
		}

		netStateMaps[v.Name] = NetTimeStampState{stat: v, timestamp: now}
		if vv, ok := n.netStates[v.Name]; ok {
			netIOUsage := calculateNetIO(netStateMaps[v.Name], vv)
			netReceiveKiBpsTimeSeries = append(netReceiveKiBpsTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "NetInterface", Value: v.Name}}, Samples: []common.Sample{{Value: netIOUsage.ReceiveKibps, Timestamp: now.Unix()}}})
			netSentKiBpsTimeSeries = append(netSentKiBpsTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "NetInterface", Value: v.Name}}, Samples: []common.Sample{{Value: netIOUsage.SentKibps, Timestamp: now.Unix()}}})
			netReceivePckpsTimeSeries = append(netReceivePckpsTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "NetInterface", Value: v.Name}}, Samples: []common.Sample{{Value: netIOUsage.ReceivePckps, Timestamp: now.Unix()}}})
			netSentPckpsTimeSeries = append(netSentPckpsTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "NetInterface", Value: v.Name}}, Samples: []common.Sample{{Value: netIOUsage.SentPckps, Timestamp: now.Unix()}}})
			netDropInTimeSeries = append(netDropInTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "NetInterface", Value: v.Name}}, Samples: []common.Sample{{Value: netIOUsage.DropIn, Timestamp: now.Unix()}}})
			netDropOutTimeSeries = append(netDropOutTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "NetInterface", Value: v.Name}}, Samples: []common.Sample{{Value: netIOUsage.DropOut, Timestamp: now.Unix()}}})
		}
	}

	n.netStates = netStateMaps

	var storeMaps = make(map[string][]common.TimeSeries, 0)
	storeMaps[string(types.MetricNetworkReceiveKiBPS)] = netReceiveKiBpsTimeSeries
	storeMaps[string(types.MetricNetworkSentKiBPS)] = netSentKiBpsTimeSeries
	storeMaps[string(types.MetricNetworkReceivePckPS)] = netReceivePckpsTimeSeries
	storeMaps[string(types.MetricNetworkSentPckPS)] = netSentPckpsTimeSeries
	storeMaps[string(types.MetricNetworkDropIn)] = netDropInTimeSeries
	storeMaps[string(types.MetricNetworkDropOut)] = netDropOutTimeSeries

	return storeMaps, nil
}

func (n *NetIOCollector) name() string {
	return netioCollectorName
}

// calculateNetIO calculate net io usage
func calculateNetIO(stat1 NetTimeStampState, stat2 NetTimeStampState) NetInterfaceUsage {

	duration := float64(stat2.timestamp.Unix() - stat1.timestamp.Unix())

	return NetInterfaceUsage{
		ReceiveKibps: float64(stat2.stat.BytesRecv-stat1.stat.BytesRecv) * 8 / 1000 / duration,
		SentKibps:    float64(stat2.stat.BytesSent-stat1.stat.BytesSent) * 8 / 1000 / duration,
		ReceivePckps: float64(stat2.stat.PacketsRecv-stat1.stat.PacketsRecv) / duration,
		SentPckps:    float64(stat2.stat.PacketsSent-stat1.stat.PacketsSent) / duration,
		DropIn:       float64(stat2.stat.Dropin-stat1.stat.Dropin) / duration,
		DropOut:      float64(stat2.stat.Dropout-stat1.stat.Dropout) / duration,
	}
}
