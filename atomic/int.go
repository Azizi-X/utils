package atomic

import "sync/atomic"

type Int32 struct {
	atomic.Int32
}

type Int64 struct {
	atomic.Int64
}

type Uint64 struct {
	atomic.Uint64
}

type Uint32 struct {
	atomic.Uint32
}
