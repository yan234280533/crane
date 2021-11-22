package nep

import (
	"sync"

	ensuranceapi "github.com/gocrane-io/api/ensurance/v1alpha1"
	"github.com/gocrane-io/crane/pkg/dsmock"
)

type CachedNodeQOSEnsurancePolicy struct {
	// The cached object of the nodeQOSEnsurancePolicy
	Nep                *ensuranceapi.NodeQOSEnsurancePolicy
	NeedStartDetection bool
	Channel            chan struct{}
	Ds                 *dsmock.DataSource
}

type NodeQOSEnsurancePolicyCache struct {
	mu     sync.Mutex // protects nepMap
	nepMap map[string]*CachedNodeQOSEnsurancePolicy
}

// ListKeys implements the interface required by DeltaFIFO to list the keys we
// already know about.
func (s *NodeQOSEnsurancePolicyCache) ListKeys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0, len(s.nepMap))
	for k := range s.nepMap {
		keys = append(keys, k)
	}
	return keys
}

// GetByKey returns the value stored in the nepMap under the given key
func (s *NodeQOSEnsurancePolicyCache) GetByKey(key string) (interface{}, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.nepMap[key]; ok {
		return v, true, nil
	}
	return nil, false, nil
}

// ListKeys implements the interface required by DeltaFIFO to list the keys we
// already know about.
func (s *NodeQOSEnsurancePolicyCache) allClusters() []*ensuranceapi.NodeQOSEnsurancePolicy {
	s.mu.Lock()
	defer s.mu.Unlock()
	neps := make([]*ensuranceapi.NodeQOSEnsurancePolicy, 0, len(s.nepMap))
	for _, v := range s.nepMap {
		neps = append(neps, v.nep)
	}
	return neps
}

func (s *NodeQOSEnsurancePolicyCache) get(name string) (*CachedNodeQOSEnsurancePolicy, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	nep, ok := s.nepMap[name]
	return nep, ok
}

func (s *NodeQOSEnsurancePolicyCache) Exist(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.nepMap[name]
	return ok
}

func (s *NodeQOSEnsurancePolicyCache) GetOrCreate(nep *ensuranceapi.NodeQOSEnsurancePolicy) *CachedNodeQOSEnsurancePolicy {
	s.mu.Lock()
	defer s.mu.Unlock()
	cacheNep, ok := s.nepMap[nep.Name]
	if !ok {
		ch := make(chan struct{})
		cacheNep = &CachedNodeQOSEnsurancePolicy{NeedStartDetection: true, Nep: nep, Channel: ch}
		s.nepMap[nep.Name] = cacheNep
	}
	return cacheNep
}

func (s *NodeQOSEnsurancePolicyCache) set(name string, nep *CachedNodeQOSEnsurancePolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nepMap[name] = nep
}

func (s *NodeQOSEnsurancePolicyCache) delete(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.nepMap, name)
}
