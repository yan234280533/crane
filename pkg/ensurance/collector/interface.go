package collector

import "sync"

type Collector interface {
	GetName() string
	Collect()
	List() sync.Map
}
