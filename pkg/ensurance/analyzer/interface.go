package analyzer

import (
	"github.com/gocrane-io/crane/pkg/ensurance/manager"
)

type Analyzer interface {
	manager.Manager
}

type AnalyzerInternal interface {
	Analyze()
}
