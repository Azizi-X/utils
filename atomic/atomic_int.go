package atomic

import "sync/atomic"

type AtomicInt struct {
	atomic.Int64
}
