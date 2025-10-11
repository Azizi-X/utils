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

type ctxInter interface {
	Alive() bool
	C() <-chan struct{}
	CancelWithReason(reason string)
	Cancel()
}

type Key string

type Event[T any] struct {
	Name      string
	Callbacks atomic.Pointer[[]callback]
}

type callback struct {
	Fn   eventInter
	Keys []Key
	Ctx  ctxInter
}

type Sync[T any] struct {
	*Event[T]
	Ctx ctxInter
}

func (s Sync[T]) Wait() {
	<-s.Ctx.C()
}

func (cb *callback) Alive() bool {
	return cb.Ctx == nil || cb.Ctx.Alive()
}

func GenKey() Key {
	return Key(fmt.Sprintf("%d", generated.Add(1)))
}

func (event *Event[T]) Remove(keys ...Key) {
	event.remove(nil, keys...)
}

func (event *Event[T]) RemoveCtx(ctx ctxInter, keys ...Key) {
	event.remove(ctx, keys...)
}

func (event *Event[T]) remove(ctx ctxInter, keys ...Key) {
	if ctx != nil {
		ctx.CancelWithReason("context canceled")
	}

	for {
		old := event.Callbacks.Load()

		updated := slices.DeleteFunc(*old, func(callback callback) bool {
			return (len(keys) > 0 && slices.Equal(callback.Keys, keys)) || (callback.Ctx != nil && callback.Ctx == ctx)
		})

		if event.Callbacks.CompareAndSwap(old, &updated) {
			break
		}
	}
}

func (event *Event[T]) SubscribeCtx(fn T, ctx ctxInter, keys ...Key) Sync[T] {
	event.addCallback(newCallback(fn, ctx, keys...))

	go func() {
		<-ctx.C()
		event.RemoveCtx(ctx, keys...)
	}()

	return Sync[T]{event, ctx}
}

func (event *Event[T]) Subscribe(fn T, keys ...Key) {
	event.addCallback(newCallback(fn, nil, keys...))
}

func (event *Event[T]) addCallback(callback callback) {
	for {
		old := event.Callbacks.Load()
		new := append(slices.Clone(*old), callback)

		if event.Callbacks.CompareAndSwap(old, &new) {
			break
		}
	}
}

func (event *Event[T]) Publish(args ...any) {
	callbacks := *event.Callbacks.Load()
	for i := range callbacks {
		if !callbacks[i].Alive() {
			continue
		}
		callbacks[i].Fn.Run(args...)
	}
}

func (event *Event[T]) Clone() *Event[T] {
	new := &Event[T]{
		Name: event.Name,
	}

	loaded := *event.Callbacks.Load()
	callbacks := slices.Clone(loaded)
	new.Callbacks.Store(&callbacks)

	return new
}

func newCallback(fn any, ctx ctxInter, keys ...Key) callback {
	return callback{
		Fn:   fn.(eventInter),
		Keys: keys,
		Ctx:  ctx,
	}
}

func New[T eventInter](name string) *Event[T] {
	event := Event[T]{
		Name: name,
	}

	initial := make([]callback, 0)
	event.Callbacks.Store(&initial)

	return &event
}

