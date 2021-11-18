package nep

import (
	"sync"

	ensuranceapi "github.com/gocrane-io/api/ensurance/v1alpha1"
)

type cachedNodeQOSEnsurancePolicy struct {
	// The cached state of the nodeQOSEnsurancePolicy
	state *ensuranceapi.NodeQOSEnsurancePolicy
}

type nodeQOSEnsurancePolicyCache struct {
	mu     sync.Mutex // protects nepMap
	nepMap map[string]*cachedNodeQOSEnsurancePolicy
}

// ListKeys implements the interface required by DeltaFIFO to list the keys we
// already know about.
func (s *nodeQOSEnsurancePolicyCache) ListKeys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0, len(s.nepMap))
	for k := range s.nepMap {
		keys = append(keys, k)
	}
	return keys
}

// GetByKey returns the value stored in the nepMap under the given key
func (s *nodeQOSEnsurancePolicyCache) GetByKey(key string) (interface{}, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.nepMap[key]; ok {
		return v, true, nil
	}
	return nil, false, nil
}

// ListKeys implements the interface required by DeltaFIFO to list the keys we
// already know about.
func (s *nodeQOSEnsurancePolicyCache) allClusters() []*ensuranceapi.NodeQOSEnsurancePolicy {
	s.mu.Lock()
	defer s.mu.Unlock()
	neps := make([]*ensuranceapi.NodeQOSEnsurancePolicy, 0, len(s.nepMap))
	for _, v := range s.nepMap {
		neps = append(neps, v.state)
	}
	return neps
}

func (s *nodeQOSEnsurancePolicyCache) get(name string) (*cachedNodeQOSEnsurancePolicy, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	nep, ok := s.nepMap[name]
	return nep, ok
}

func (s *nodeQOSEnsurancePolicyCache) Exist(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.nepMap[name]
	return ok
}

func (s *nodeQOSEnsurancePolicyCache) getOrCreate(name string) *cachedNodeQOSEnsurancePolicy {
	s.mu.Lock()
	defer s.mu.Unlock()
	nep, ok := s.nepMap[name]
	if !ok {
		nep = &cachedNodeQOSEnsurancePolicy{}
		s.nepMap[name] = nep
	}
	return nep
}

func (s *nodeQOSEnsurancePolicyCache) set(name string, nep *cachedNodeQOSEnsurancePolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nepMap[name] = nep
}

func (s *nodeQOSEnsurancePolicyCache) delete(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.nepMap, name)
}
