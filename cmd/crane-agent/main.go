package main

import (
	"fmt"
	"os"

	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/logs"

	"github.com/gocrane-io/crane/cmd/crane-agent/app"
	"github.com/gocrane-io/crane/pkg/utils/clogs"
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
