package statestore

import (
	"fmt"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/gocrane-io/crane/pkg/ensurance/statestore/nodelocal"
	"github.com/gocrane-io/crane/pkg/ensurance/statestore/types"
	"github.com/gocrane-io/crane/pkg/utils/clogs"
	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
)

type stateStoreManager struct {
	nepInformer cache.SharedIndexInformer

	eventChannel chan types.UpdateEvent
	index        uint64
	configCache  sync.Map

	collectors  []collector
	StatusCache sync.Map
}

func NewStateStoreManager(nepInformer cache.SharedIndexInformer) StateStore {
	var eventChan = make(chan types.UpdateEvent)
	return &stateStoreManager{nepInformer: nepInformer, eventChannel: eventChan}
}

func (s *stateStoreManager) Name() string {
	return "StateStoreManager"
}

func (s *stateStoreManager) Run(stop <-chan struct{}) {

	// check need to update config
	go func() {
		updateTicker := time.NewTicker(10 * time.Second)
		defer updateTicker.Stop()
		for {
			select {
			case <-updateTicker.C:
				clogs.Log().Info("StateStore config check run periodically")
				if s.checkConfig() {
					s.index++
					s.eventChannel <- types.UpdateEvent{Index: s.index}
				}
				return
			case <-stop:
				clogs.Log().Info("StateStore config check exit")
				return
			}
		}

	}()

	// do collect periodically
	go func() {
		updateTicker := time.NewTicker(10 * time.Second)
		defer updateTicker.Stop()
		for {
			select {
			case <-updateTicker.C:
				clogs.Log().V(2).Info("StateStore run periodically")
				for _, c := range s.collectors {
					if data, err := c.Collect(); err == nil {
						for key, v := range data {
							s.StatusCache.Store(key, v)
						}
					} else {
						clogs.Log().Error(err, "StateStore collect failed", c.GetType())
					}
				}
				return
			case v := <-s.eventChannel:
				clogs.Log().V(3).Info("StateStore update config index", v.Index)
				s.updateConfig()
				return
			case <-stop:
				clogs.Log().V(2).Info("StateStore exit")
				return
			}
		}
	}()

	return
}

func (s *stateStoreManager) List() sync.Map {
	return s.StatusCache
}

func (s *stateStoreManager) AddMetric(key string, t types.CollectType, metricName string, Selector *metav1.LabelSelector) error {
	if t != types.NodeLocalCollectorType {
		return fmt.Errorf("only support node local collect")
	}

	if !nodelocal.CheckMetricNameExist(types.MetricName(metricName)) {
		return fmt.Errorf("node local not support metric name %s", metricName)
	}

	return nil
}

func (s *stateStoreManager) DeleteMetric(key string, t types.CollectType) {
	return
}

func (s *stateStoreManager) checkConfig() bool {
	// step1 copy neps
	var neps []*ensuranceapi.NodeQOSEnsurancePolicy
	allNeps := s.nepInformer.GetStore().List()
	for _, n := range allNeps {
		nep := n.(*ensuranceapi.NodeQOSEnsurancePolicy).DeepCopy()
		if nep.Spec.NodeQualityProbe.Handler.NodeLocalGet == nil {
			clogs.Log().V(4).Info("Warning: skip the config not node-local, it will support other kind of config in the future")
			continue
		}

		neps = append(neps, nep)
	}

	// step2 check it needs to update
	var nodeLocal bool
	for _, n := range neps {
		if n.Spec.NodeQualityProbe.Handler.NodeLocalGet != nil {
			nodeLocal = true
			if _, ok := s.configCache.Load(string(types.NodeLocalCollectorType)); !ok {
				return true
			}
		}
	}

	if !nodeLocal {
		if _, ok := s.configCache.Load(string(types.NodeLocalCollectorType)); ok {
			return true
		}
	}

	return false
}

func (s *stateStoreManager) updateConfig() {
	// step1 copy neps
	var neps []*ensuranceapi.NodeQOSEnsurancePolicy
	allNeps := s.nepInformer.GetStore().List()
	for _, n := range allNeps {
		nep := n.(*ensuranceapi.NodeQOSEnsurancePolicy).DeepCopy()
		if nep.Spec.NodeQualityProbe.Handler.NodeLocalGet == nil {
			clogs.Log().V(4).Info("Warning: skip the config not node-local, it will support other kind of config in the future")
			continue
		}

		neps = append(neps, nep)
	}

	// step2 update the config
	var nodeLocal bool
	for _, n := range neps {
		if n.Spec.NodeQualityProbe.Handler.NodeLocalGet != nil {
			nodeLocal = true
			if _, ok := s.configCache.Load(string(types.NodeLocalCollectorType)); !ok {
				nc := nodelocal.NewNodeLocal()
				s.collectors = append(s.collectors, nc)
				s.configCache.Store(string(types.NodeLocalCollectorType), types.MetricNameConfigs{})
			}
		}
	}

	if !nodeLocal {
		if _, ok := s.configCache.Load(string(types.NodeLocalCollectorType)); ok {
			s.configCache.Delete(string(types.NodeLocalCollectorType))
			var collectors []collector
			for _, c := range s.collectors {
				if c.GetType() != types.NodeLocalCollectorType {
					collectors = append(collectors, c)
				}
			}
			s.collectors = collectors
		}
	}

	return

}
