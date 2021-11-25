package statestore

import (
	"github.com/gocrane-io/crane/pkg/ensurance/manager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sync"
)

type StateStore interface {
	manager.Manager
	List() sync.Map
	AddMetric(key string, metricName string, Selector *metav1.LabelSelector)
	DeleteMetric(key string)
}
