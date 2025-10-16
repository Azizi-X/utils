package atomic

import "sync/atomic"

type AtomicString struct {
	_ nocmp
	atomic.Value
}

func NewString(val string) *AtomicString {
	str := &AtomicString{}
	if val != "" {
		str.Store(val)
	}
	return str
}

func (str *AtomicString) Load() string {
	return unpackString(str.Value.Load())
}

func (str *AtomicString) Store(val string) {
	str.Value.Store(packString(val))
}

func (str *AtomicString) CompareAndSwap(old, new string) (swapped bool) {
	if str.Value.CompareAndSwap(packString(old), packString(new)) {
		return true
	}

	if old == "" {
		return str.Value.CompareAndSwap(nil, packString(new))
	}

	return false
}

func (str *AtomicString) Swap(val string) (old string) {
	return unpackString(str.Value.Swap(packString(val)))
}
