//go:build !linux
// +build !linux

package cadvisor

import (
	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
	info "github.com/google/cadvisor/info/v1"
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type CadvisorCollector struct {
	Manager Manager
}

type Manager struct {
}

func NewCadvisor(_ corelisters.PodLister) *CadvisorCollector {
	return &CadvisorCollector{}
}

func (c *CadvisorCollector) Stop() error {
	return nil
}

func (c *CadvisorCollector) GetType() types.CollectType {
	return types.CadvisorCollectorType
}

func (c *CadvisorCollector) Collect() (map[string][]common.TimeSeries, error) {
	return nil, nil
}

func CheckMetricNameExist(name string) bool {
	return false
}

func (m *Manager) ContainerInfoV2(containerName string, options cadvisorapiv2.RequestOptions) (map[string]cadvisorapiv2.ContainerInfo, error) {
	return nil, nil
}

func (m *Manager) GetContainerInfo(containerName string, query *info.ContainerInfoRequest) (*info.ContainerInfo, error) {
	return nil, nil
}
