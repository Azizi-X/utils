package atomic

import "sync/atomic"

type Int struct {
	atomic.Int64
}

type Int32 struct {
	atomic.Int32
}
