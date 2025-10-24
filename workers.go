package utils

import "sync"

type Worker[T any] struct {
	ch   chan T
	fn   func(T)
	once sync.Once
}

func (w *Worker[T]) Close() {
	w.once.Do(func() {
		close(w.ch)
	})
}

func (w *Worker[T]) Worker() {
	for event := range w.ch {
		w.fn(event)
	}
}

func (w *Worker[T]) Send(v T) {
	w.ch <- v
}

func NewWorker[T any](workers, buf int, fn func(T)) *Worker[T] {
	ch := make(chan T, buf)

	handler := Worker[T]{
		ch: ch,
		fn: fn,
	}

	for range workers {
		go handler.Worker()
	}

	return &handler
}
