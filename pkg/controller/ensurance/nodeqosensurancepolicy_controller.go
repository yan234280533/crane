/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"time"

	"github.com/gocrane-io/crane/pkg/dsmock"
	"github.com/gocrane-io/crane/pkg/ensurance/opamock"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ensuranceapi "github.com/gocrane-io/api/ensurance/v1alpha1"
	"github.com/gocrane-io/crane/pkg/ensurance/cache"
	"github.com/gocrane-io/crane/pkg/ensurance/nep"
)

// NodeQOSEnsurancePolicyController reconciles a NodeQOSEnsurancePolicy object
type NodeQOSEnsurancePolicyController struct {
	client.Client
	Scheme         *runtime.Scheme
	Log            logr.Logger
	RestMapper     meta.RESTMapper
	Recorder       record.EventRecorder
	Cache          *nep.NodeQOSEnsurancePolicyCache
	DetectionCache *cache.DetectionConditionCache
}

//+kubebuilder:rbac:groups=ensurance.crane.io.crane.io,resources=nodeqosensurancepolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ensurance.crane.io.crane.io,resources=nodeqosensurancepolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ensurance.crane.io.crane.io,resources=nodeqosensurancepolicies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NodeQOSEnsurancePolicy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (c *NodeQOSEnsurancePolicyController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	c.Log.Info("got", "nep", req.NamespacedName)

	nep := &ensuranceapi.NodeQOSEnsurancePolicy{}
	if err := c.Client.Get(ctx, req.NamespacedName, nep); err != nil {
		// The resource may be deleted, in this case we need to stop the processing.
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{Requeue: true}, err
	}

	return c.reconcileNep(nep)
}

// SetupWithManager sets up the controller with the Manager.
func (c *NodeQOSEnsurancePolicyController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ensuranceapi.NodeQOSEnsurancePolicy{}).
		Complete(c)
}

func (c *NodeQOSEnsurancePolicyController) reconcileNep(nep *ensuranceapi.NodeQOSEnsurancePolicy) (ctrl.Result, error) {
	var cachedNep = c.Cache.GetOrCreate(nep)

	if cachedNep.NeedStartDetection {
		ds, err := dsmock.GetDataSourceFromNodeProbes(nep.Spec.NodeQualityProbe)
		if err != nil {
			return ctrl.Result{}, err
		}

		cachedNep.Ds = ds

		if err := c.startDetection(cachedNep); err != nil {
			return ctrl.Result{}, err
		}
		cachedNep.NeedStartDetection = false
	}

	return ctrl.Result{}, nil
}

func (c *NodeQOSEnsurancePolicyController) startDetection(cachedNep *nep.CachedNodeQOSEnsurancePolicy) error {

	for _, v := range cachedNep.Nep.Spec.ObjectiveEnsurance {
		var detection = cache.DetectionCondition{PolicyName: cachedNep.Nep.Name, Namespace: cachedNep.Nep.Namespace, ObjectiveEnsuranceName: v.AvoidanceActionName}
		go wait.PollImmediateUntil(time.Duration(int64(cachedNep.Nep.Spec.NodeQualityProbe.PeriodSeconds))*time.Second, doDetection(cachedNep.Ds, c.DetectionCache, v, detection), cachedNep.Channel)
	}

	return nil
}

func doDetection(ds *dsmock.DataSource, dCache *cache.DetectionConditionCache, object ensuranceapi.ObjectiveEnsurance, detection cache.DetectionCondition) func() (bool, error) {

	return func() (bool, error) {
		//step1: get metrics
		value, err := ds.LatestTimeSeries(object.MetricRule.Metric.Name, object.MetricRule.Metric.Selector)
		if err != nil {
			return false, err
		}

		//step2: use opa to check if reached
		opamock.OpaEval(object.MetricRule.Metric.Name, float64(object.MetricRule.Target.Value.Value()), value)

		//step3: check is reached action or restored, set the detection

		//step4: update the cache
		UpdateDetectionIfNeed(dCache, detection)

		return true, nil
	}
}

func UpdateDetectionIfNeed(dCache *cache.DetectionConditionCache, detection cache.DetectionCondition) error {
	// step1: get detection object from cache
	detectionOld, ok := dCache.Get(cache.GenerateDetectionKey(detection))
	if !ok {
		dCache.Set(detection)
		return nil
	}

	// it has no changed, skip update
	if (detection.Triggered == detectionOld.Triggered) && (detection.Restored == detectionOld.Restored) {
		return nil
	}

	//update the detection
	dCache.Set(detection)

	return nil
}
