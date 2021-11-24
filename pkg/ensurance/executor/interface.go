package executor

type Detector interface {
	Avoid()
	Restore()
}

