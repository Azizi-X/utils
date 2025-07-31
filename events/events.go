package Events

import (
	"fmt"
	"slices"
	"sync/atomic"
)

var generated atomic.Uint64

type eventInter interface {
	Run(args ...any)
}

type Key string

type callback struct {
	Fn   eventInter
	Keys []Key
}

type Event[T any] struct {
	Name      string
	Callbacks atomic.Pointer[[]callback]
}

func GenKey() Key {
	return Key(fmt.Sprintf("%d", generated.Add(1)))
}

func (event *Event[T]) Remove(keys ...Key) {
	for {
		old := event.Callbacks.Load()

		updated := slices.DeleteFunc(*old, func(callback callback) bool {
			return slices.Equal(callback.Keys, keys)
		})

		if event.Callbacks.CompareAndSwap(old, &updated) {
			break
		}
	}
}

func (event *Event[T]) Subscribe(fn T, keys ...Key) {
	for {
		old := event.Callbacks.Load()
		new := append(slices.Clone(*old), newCallback(fn, keys...))

		if event.Callbacks.CompareAndSwap(old, &new) {
			break
		}
	}
}

func (event *Event[T]) Publish(args ...any) {
	callbacks := *event.Callbacks.Load()
	for i := range callbacks {
		callbacks[i].Fn.Run(args...)
	}
}

func newCallback(fn any, keys ...Key) callback {
	return callback{
		Fn:   fn.(eventInter),
		Keys: keys,
	}
}

func New[T any](name string) *Event[T] {
	event := Event[T]{
		Name: name,
	}

	initial := make([]callback, 0)
	event.Callbacks.Store(&initial)

	return &event
}
