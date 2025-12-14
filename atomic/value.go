package atomic

import "sync/atomic"

type V[T any] struct {
	_ nocmp
	atomic.Value
}

func (z *V[T]) Load() T {
	return unpackV[T](z.Value.Load())
}

func (z *V[T]) Store(val T) {
	z.Value.Store(packV(val))
}

func (z *V[T]) CompareAndSwap(old, new T) (swapped bool) {
	return z.Value.CompareAndSwap(packV(old), packV(new))
}

func (z *V[T]) Swap(val T) (old T) {
	return unpackV[T](z.Value.Swap(packV(val)))
}
