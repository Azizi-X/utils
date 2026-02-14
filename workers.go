package utils

import "sync"

type Worker[T any] struct {
	ch   chan T
	fn   func(T)
	once sync.Once
	wg   sync.WaitGroup
}

func (w *Worker[T]) Close() {
	w.once.Do(func() {
		close(w.ch)
		w.wg.Wait()
	})
}

func (w *Worker[T]) Worker() {
	defer w.wg.Done()

	for event := range w.ch {
		w.fn(event)
	}
}

func (w *Worker[T]) Send(v T) {
	w.ch <- v
}

func NewWorker[T any](workers, buf int, fn func(T)) *Worker[T] {
	if fn == nil {
		panic("fn can not be nil")
	} else if workers <= 0 {
		panic("workers must be greater than 0")
	}

	ch := make(chan T, buf)

	handler := Worker[T]{
		ch: ch,
		fn: fn,
	}

	handler.wg.Add(workers)

	for range workers {
		go handler.Worker()
	}

	return &handler
}
