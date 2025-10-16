package atomic

import "sync/atomic"

type String struct {
	_ nocmp
	atomic.Value
}

func NewString(v string) *String {
	str := &String{}
	if val != "" {
		str.Store(v)
	}
	return str
}

func (str *String) Load() string {
	return unpackString(str.Value.Load())
}

func (str *String) Store(v string) {
	str.Value.Store(packString(v))
}

func (str *String) CompareAndSwap(old, new string) (swapped bool) {
	if str.Value.CompareAndSwap(packString(old), packString(new)) {
		return true
	}

	if old == "" {
		return str.Value.CompareAndSwap(nil, packString(new))
	}

	return false
}

func (str *String) Swap(v string) (old string) {
	return unpackString(str.Value.Swap(packString(v)))
}
