package statestore

import (
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gocrane-io/crane/pkg/ensurance/manager"
	"github.com/gocrane-io/crane/pkg/ensurance/statestore/types"
	"github.com/gocrane-io/crane/pkg/utils"
)

type StateStore interface {
	manager.Manager
	List() sync.Map
	AddMetric(key string, t types.CollectType, metricName string, Selector *metav1.LabelSelector) error
	DeleteMetric(key string, t types.CollectType)
}

type collector interface {
	GetType() types.CollectType
	Collect() (map[string]utils.TimeSeries, error)
}
