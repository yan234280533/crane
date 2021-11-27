package analyzer

import (
	"fmt"
	"strings"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	ecache "github.com/gocrane-io/crane/pkg/ensurance/cache"
	"github.com/gocrane-io/crane/pkg/ensurance/executor"
	"github.com/gocrane-io/crane/pkg/ensurance/logic"
	"github.com/gocrane-io/crane/pkg/utils"
	"github.com/gocrane-io/crane/pkg/utils/clogs"
	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
)

type AnalyzerManager struct {
	podInformer       cache.SharedIndexInformer
	nodeInformer      cache.SharedIndexInformer
	nepInformer       cache.SharedIndexInformer
	avoidanceInformer cache.SharedIndexInformer
	recorder          record.EventRecorder
	noticeCh          chan<- executor.AvoidanceExecutor

	logic             logic.Logic
	nodeStatus        sync.Map
	lastTriggeredTime time.Time
	dcsOlder          []ecache.DetectionCondition
	acsOlder          executor.AvoidanceExecutor
}

// AnalyzerManager create analyzer manager
func NewAnalyzerManager(podInformer cache.SharedIndexInformer, nodeInformer cache.SharedIndexInformer, nepInformer cache.SharedIndexInformer,
	avoidanceInformer cache.SharedIndexInformer, record record.EventRecorder, noticeCh chan<- executor.AvoidanceExecutor) Analyzer {

	opaLogic := logic.NewOpaLogic()

	return &AnalyzerManager{
		logic:             opaLogic,
		noticeCh:          noticeCh,
		recorder:          record,
		podInformer:       podInformer,
		nodeInformer:      nodeInformer,
		nepInformer:       nepInformer,
		avoidanceInformer: avoidanceInformer,
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
				clogs.Log().V(2).Info("Analyzer run periodically")
				s.Analyze()
			case <-stop:
				clogs.Log().V(2).Info("Analyzer exit")
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

	var asMaps map[string]*ensuranceapi.AvoidanceAction
	allAss := s.avoidanceInformer.GetStore().List()
	for _, n := range allAss {
		as := n.(*ensuranceapi.AvoidanceAction)
		asMaps[as.Name] = as
	}

	// step 2: do analyze for neps
	var dcs []ecache.DetectionCondition
	for _, n := range neps {
		for _, v := range n.Spec.ObjectiveEnsurances {
			detection, err := s.doAnalyze(v)
			if err != nil {
				//warning and continue
			}
			detection.Nep = n
			dcs = append(dcs, detection)
		}
	}

	//step 3 : doMerge
	avoidanceAction, err := s.doMerge(neps, asMaps, dcs)
	if err != nil {
		// to return err
	}

	//step 4 :notice the avoidance manager
	s.noticeAvoidanceManager(avoidanceAction)

	return
}

func (s *AnalyzerManager) doAnalyze(object ensuranceapi.ObjectiveEnsurance) (ecache.DetectionCondition, error) {
	//step1: get metric value
	value, err := s.getMetricFromMap(object.MetricRule.Metric.Name, object.MetricRule.Metric.Selector)
	if err != nil {
		return ecache.DetectionCondition{}, err
	}

	//step2: use opa to check if reached
	s.logic.EvalWithMetric(object.MetricRule.Metric.Name, float64(object.MetricRule.Target.Value.Value()), value)

	//step3: check is reached action or restored, set the detection

	return ecache.DetectionCondition{DryRun: object.DryRun, ActionName: object.AvoidanceActionName}, nil
}

func (s *AnalyzerManager) doMerge(neps []*ensuranceapi.NodeQOSEnsurancePolicy, asMaps map[string]*ensuranceapi.AvoidanceAction, dcs []ecache.DetectionCondition) (executor.AvoidanceExecutor, error) {
	//step1 filter the only dryRun detection
	var dcsFiltered []ecache.DetectionCondition
	for _, dc := range dcs {
		if !dc.DryRun {
			s.doLogEvent(dc)
		} else {
			dcsFiltered = append(dcsFiltered, dc)
		}
	}

	var ae executor.AvoidanceExecutor

	//step2 do BlockScheduled merge
	var bBlockScheduled bool
	var bRestoreScheduled bool
	for _, dc := range dcsFiltered {
		if dc.Triggered {
			bBlockScheduled = true
			bRestoreScheduled = false
		}
	}

	var now = time.Now()
	if bBlockScheduled {
		ae.BlockScheduledExecutor.BlockScheduledQOSPriority = &executor.ScheduledQOSPriority{PodQOSClass: v1.PodQOSBestEffort, PriorityClassValue: 0}
		s.lastTriggeredTime = now
	}

	if bRestoreScheduled {
		bRestoreScheduled = true
		for _, dc := range dcsFiltered {
			action, ok := asMaps[dc.ActionName]
			if !ok {
				clogs.Log().V(4).Info("Waring: doMerge for detection the action ", dc.ActionName, " not found")
				continue
			}

			if dc.Restored {
				var schedulingCoolDown = utils.GetInt64withDefault(action.Spec.SchedulingCoolDown, 300)

				if !now.After(s.lastTriggeredTime.Add(-time.Duration(schedulingCoolDown) * time.Second)) {
					bRestoreScheduled = false
					break
				}
			}
		}
	}

	if bRestoreScheduled {
		ae.BlockScheduledExecutor.RestoreScheduledQOSPriority = &executor.ScheduledQOSPriority{PodQOSClass: v1.PodQOSBestEffort, PriorityClassValue: 0}
	}

	//step3 do Throttle merge FilterAndSortThrottlePods
	//step4 do Evict merge  FilterAndSortEvictPods
	return ae, nil
}

func (s *AnalyzerManager) doLogEvent(dc ecache.DetectionCondition) {

	var key = strings.Join([]string{dc.Nep.Name, dc.ObjectiveEnsuranceName}, "/")

	//step1 print log if the detection state is changed
	//step2 produce event
	if dc.Triggered {
		clogs.Log().Info(fmt.Sprintf("%s triggered action %s", key, dc.ActionName))
		// record an event about the objective ensurance triggered
		s.recorder.Event(dc.Nep, v1.EventTypeWarning, "ObjectiveEnsuranceTriggered", fmt.Sprintf("%s triggered action %s", key, dc.ActionName))
	}

	if dc.Restored {
		clogs.Log().Info(fmt.Sprintf("%s restored action %s", key, dc.ActionName))
		// record an event about the objective ensurance restored
		s.recorder.Event(dc.Nep, v1.EventTypeWarning, "ObjectiveEnsuranceRestored", fmt.Sprintf("%s restored action %s", key, dc.ActionName))
	}

	return
}

func (s *AnalyzerManager) getMetricFromMap(metricName string, selector *metav1.LabelSelector) (float64, error) {
	// step1: generate the key for the metric
	// step2: get the value from map
	return 0.0, nil
}

func (s *AnalyzerManager) noticeAvoidanceManager(as executor.AvoidanceExecutor) {
	//step1: check need to notice avoidance manager

	//step2: notice by channel
	s.noticeCh <- as
	return
}
