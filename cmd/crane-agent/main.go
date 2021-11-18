package main

import (
	"fmt"
	"github.com/gocrane-io/crane/pkg/utils/clogs"
	"os"

	genericapiserver "k8s.io/apiserver/pkg/server"

	"github.com/gocrane-io/crane/cmd/crane-agent/app"
	"k8s.io/component-base/logs"
)

// crane-agent main.
func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	clogs.InitLogs("crane-manager")

	ctx := genericapiserver.SetupSignalContext()

	if err := app.NewManagerCommand(ctx).Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
