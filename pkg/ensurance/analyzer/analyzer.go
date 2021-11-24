package analyzer

import (
	ensuranceapi "github.com/gocrane-io/api/ensurance/v1alpha1"
	"github.com/gocrane-io/crane/pkg/dsmock"
	ecache "github.com/gocrane-io/crane/pkg/ensurance/cache"
	"github.com/gocrane-io/crane/pkg/ensurance/opamock"
	"github.com/gocrane-io/crane/pkg/utils/clogs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"sync"
	"time"
)

type AnalyzerManager struct {
	podInformer  cache.SharedIndexInformer
	nodeInformer cache.SharedIndexInformer
	nepInformer  cache.SharedIndexInformer

	NodeStatus sync.Map
	dcsOlder   []ecache.DetectionCondition
	acsOlder   AvoidanceActionStruct
}

// AnalyzeManager create analyzer manager
func NewAnalyzerManager(podInformer cache.SharedIndexInformer, nodeInformer cache.SharedIndexInformer, nepInformer cache.SharedIndexInformer) *AnalyzerManager {
	return &AnalyzerManager{
		podInformer:  podInformer,
		nodeInformer: nodeInformer,
		nepInformer:  nepInformer,
	}
}

func (s *AnalyzerManager) Name() string {
	return "AnalyzeManager"
}

func (s *AnalyzerManager) Run(stop <-chan struct{}) {
	go func() {
		updateTicker := time.NewTicker(10 * time.Second)
		defer updateTicker.Stop()
		for {
			select {
			case <-updateTicker.C:
				clogs.Log().Info("Analyzer run periodically")
				s.Analyze()
			case <-stop:
				clogs.Log().Info("Analyzer exit")
				return
			}
		}
	}()

	return
}

func (s *AnalyzerManager) Analyze() {
	// step1 copy neps
	var neps []*ensuranceapi.NodeQOSEnsurancePolicy
	allNeps := s.nepInformer.GetStore().List()
	for _, n := range allNeps {
		nep := n.(*ensuranceapi.NodeQOSEnsurancePolicy)
		neps = append(neps, nep.DeepCopy())
	}

	// step 2: do analyze for neps
	var dcs []ecache.DetectionCondition
	for _, n := range neps {
		for _, v := range n.Spec.ObjectiveEnsurance {
			detection, err := s.doAnalyze(v)
			if err != nil {
				//warning and continue
			}
			detection.PolicyName = n.Name
			detection.Namespace = n.Namespace
			detection.ObjectiveEnsuranceName = v.AvoidanceActionName
			dcs = append(dcs, detection)
		}
	}

	//step 3 : doMerge
	avoidanceActionStruct, err := s.doMerge(dcs)
	if err != nil {

	}

	//step 4 :notice the avoidance manager
	s.noticeAvoidanceManager(avoidanceActionStruct)

	return
}

func (s *AnalyzerManager) doAnalyze(object ensuranceapi.ObjectiveEnsurance) (ecache.DetectionCondition, error) {
	//step1: get metric value
	value, err := s.getValueFromMap(object.MetricRule.Metric.Name, object.MetricRule.Metric.Selector)
	if err != nil {
		return ecache.DetectionCondition{}, err
	}

	//step2: use opa to check if reached
	opamock.OpaEval(object.MetricRule.Metric.Name, float64(object.MetricRule.Target.Value.Value()), value)

	//step3: check is reached action or restored, set the detection

	return ecache.DetectionCondition{}, nil
}

func (s *AnalyzerManager) doMerge(dcs []ecache.DetectionCondition) (AvoidanceActionStruct, error) {
	//step1 filter the only dryRun detection
	//step2 do BlockScheduled merge
	//step3 do Throttle merge FilterAndSortThrottlePods
	//step3 do Evict merge  FilterAndSortEvictPods
	return AvoidanceActionStruct{}, nil
}

func (s *AnalyzerManager) getValueFromMap(metricName string, selector *metav1.LabelSelector) (float64, error) {
	// step1: generate the key for the metric
	// step2: get the value from map
	return 0.0, nil
}

func (s *AnalyzerManager) noticeAvoidanceManager(as AvoidanceActionStruct) bool {
	return false
}
