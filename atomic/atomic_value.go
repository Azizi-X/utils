package atomic

import "sync/atomic"

type AtomicV[T any] struct {
	_ nocmp
	atomic.Value
}

func (z *AtomicV[T]) Load() T {
	return unpackV[T](z.Value.Load())
}

func (z *AtomicV[T]) Store(val T) {
	z.Value.Store(packV(val))
}

func (z *AtomicV[T]) CompareAndSwap(old, new T) (swapped bool) {
	return z.Value.CompareAndSwap(packV(old), packV(new))
}

func (z *AtomicV[T]) Swap(val T) (old T) {
	return unpackV[T](z.Value.Swap(packV(val)))
}
