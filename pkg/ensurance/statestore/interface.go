package statestore

import (
	"github.com/gocrane-io/crane/pkg/ensurance/manager"
	"sync"
)

type StateStore interface {
	manager.Manager
	List() sync.Map
}
