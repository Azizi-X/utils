package atomic

import "sync/atomic"

type Bool struct {
	atomic.Bool
}
