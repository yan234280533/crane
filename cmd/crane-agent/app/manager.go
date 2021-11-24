package app

import (
	"context"
	"flag"
	"fmt"
	"github.com/gocrane-io/crane/pkg/ensurance/analyzer"
	"github.com/gocrane-io/crane/pkg/ensurance/avoidance"
	"github.com/gocrane-io/crane/pkg/ensurance/cache"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	ensuaranceapi "github.com/gocrane-io/api/ensurance/v1alpha1"
	ensuaranceset "github.com/gocrane-io/api/pkg/generated/clientset/versioned"
	"github.com/gocrane-io/crane/cmd/crane-agent/app/options"
	ensurancecontroller "github.com/gocrane-io/crane/pkg/controller/ensurance"
	einformer "github.com/gocrane-io/crane/pkg/ensurance/informer"
	"github.com/gocrane-io/crane/pkg/ensurance/nep"
	"github.com/gocrane-io/crane/pkg/utils/clogs"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(ensuaranceapi.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

// NewManagerCommand creates a *cobra.Command object with default parameters
func NewManagerCommand(ctx context.Context) *cobra.Command {
	opts := options.NewOptions()

	cmd := &cobra.Command{
		Use:  "crane-agent",
		Long: `The crane agent is responsible agent in crane`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(); err != nil {
				clogs.Log().Error(err, "opts complete failed,exit")
				os.Exit(255)
			}
			if err := opts.Validate(); err != nil {
				clogs.Log().Error(err, "opts validate failed,exit")
				os.Exit(255)
			}

			if err := Run(ctx, opts); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	opts.AddFlags(cmd.Flags())
	return cmd
}

// Run runs the crane-agent with options. This should never exit.
func Run(ctx context.Context, opts *options.Options) error {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     opts.MetricsAddr,
		HealthProbeBindAddress: opts.BindAddr,
		Port:                   int(opts.WebhookPort),
		Host:                   opts.WebhookHost,
		LeaderElection:         false,
	})
	if err != nil {
		clogs.Log().Error(err, "unable to start crane agent")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		clogs.Log().Error(err, "failed to add health check endpoint")
		return err
	}

	initializationControllers(mgr, opts)

	clogs.Log().Info("Starting crane agent")
	if err := mgr.Start(ctx); err != nil {
		clogs.Log().Error(err, "problem running crane manager")
		return err
	}

	return nil
}

// initializationControllers setup controllers with manager
func initializationControllers(mgr ctrl.Manager, opts *options.Options) {
	clogs.Log().Info(fmt.Sprintf("opts %v", opts))

	nepRecorder := mgr.GetEventRecorderFor("node-qos-controller")

	var nodeDetectionCache = cache.DetectionConditionCache{}

	if err := (&ensurancecontroller.NodeQOSEnsurancePolicyController{
		Client:         mgr.GetClient(),
		Log:            clogs.Log().WithName("node-qos-controller"),
		Scheme:         mgr.GetScheme(),
		RestMapper:     mgr.GetRESTMapper(),
		Recorder:       nepRecorder,
		Cache:          &nep.NodeQOSEnsurancePolicyCache{},
		DetectionCache: &nodeDetectionCache,
	}).SetupWithManager(mgr); err != nil {
		clogs.Log().Error(err, "unable to create controller", "controller", "NodeQOSEnsurancePolicyController")
		os.Exit(1)
	}

	//New avoidance manager
	generatedClient := kubernetes.NewForConfigOrDie(mgr.GetConfig())
	clientSet := ensuaranceset.NewForConfigOrDie(mgr.GetConfig())

	ctx := einformer.NewContextInitWithClient(generatedClient, clientSet, opts.HostnameOverride)
	podInformer := ctx.GetPodFactory().Core().V1().Pods().Informer()
	nodeInformer := ctx.GetNodeFactory().Core().V1().Nodes().Informer()
	avoidanceInformer := ctx.GetAvoidanceFactory().Ensurance().V1alpha1().AvoidanceActions().Informer()
	nepInformer := ctx.GetAvoidanceFactory().Ensurance().V1alpha1().NodeQOSEnsurancePolicies().Informer()

	stopChannel := make(chan struct{})
	ctx.Run(stopChannel)

	avoidanceManager := avoidance.NewAvoidanceManager(podInformer, nodeInformer, avoidanceInformer)
	avoidanceManager.Run(stopChannel)

	analyzer.NewAnalyzerManager(podInformer, nodeInformer, nepInformer)

	return
}
