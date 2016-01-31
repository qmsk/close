package workers

import (
    "close/worker"
)

var Options worker.Options

func init() {
    Options.Register("dummyworker", &DummyConfig{})
}
