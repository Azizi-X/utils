package atomic

import "sync/atomic"

type Int struct {
	atomic.Int64
}
