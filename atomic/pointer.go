package atomic

import "sync/atomic"

type P[T any] struct {
	_ nocmp
	atomic.Pointer[T]
}
