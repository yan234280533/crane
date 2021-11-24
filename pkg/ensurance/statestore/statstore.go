package statestore

import (
	"github.com/gocrane-io/crane/pkg/ensurance/statestore/collect"
	"github.com/gocrane-io/crane/pkg/utils/clogs"
	"sync"
	"time"
)

type stateStoreManager struct {
	collectors []collect.Collector
}

func NewStateStoreManager() StateStore {
	e := collect.NewEBPF()
	n := collect.NewNodeLocal()
	m := collect.NewMetricsServer()

	collectors := []collect.Collector{e, n, m}

	return &stateStoreManager{collectors: collectors}
}

func (s *stateStoreManager) Name() string {
	return "StateStoreManager"
}

func (s *stateStoreManager) Run(stop <-chan struct{}) {
	go func() {
		updateTicker := time.NewTicker(10 * time.Second)
		defer updateTicker.Stop()
		for {
			select {
			case <-updateTicker.C:
				clogs.Log().Info("StateStore run periodically")
				for _, c := range s.collectors {
					c.Collect()
				}
			case <-stop:
				clogs.Log().Info("StateStore exit")
				return
			}
		}
	}()

	return
}

func (s *stateStoreManager) List() sync.Map {
	// for each collect to get status
	return sync.Map{}
}
