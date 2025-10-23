package Events

import (
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
	"unsafe"
)

var ErrCanceled = errors.New("context canceled")

type Key string

type eventInter[T any] interface {
	Run(args T)
}

type Event[T any, Z eventInter[T]] struct {
	Name      string
	Callbacks atomic.Pointer[[]callback[T, Z]]
}

type callback[T any, Z eventInter[T]] struct {
	Callback Z
	Ctx      ctxInter
	Keys     []Key
}

type ctxInter interface {
	Alive() bool
	Wait() bool
	CancelWithErr(error)
}

type Sync struct {
	ctxInter
}

func New[T any, Z eventInter[T]](name string) *Event[T, Z] {
	event := &Event[T, Z]{
		Name: name,
	}

	initial := make([]callback[T, Z], 0)
	event.Callbacks.Store(&initial)

	return event
}

func (cb *callback[T, Z]) Alive() bool {
	return cb.Ctx == nil || cb.Ctx.Alive()
}

func (cb *callback[T, Z]) isNil() bool {
	return (*struct {
		Type *struct{}
	})(unsafe.Pointer(&cb.Callback)).Type == nil
}

func (event *Event[T, Z]) Clone() *Event[T, Z] {
	new := &Event[T, Z]{
		Name: event.Name,
	}

	loaded := *event.Callbacks.Load()
	callbacks := slices.Clone(loaded)
	new.Callbacks.Store(&callbacks)

	return new
}

func (e *Event[T, Z]) Subscribe(fn Z, keys ...Key) {
	e.SubscribeIdx(fn, -1, keys...)
}

func (e *Event[T, Z]) SubscribeIdx(fn Z, idx int, keys ...Key) {
	e.addCallback(idx, fn, nil, keys...)
}

func (e *Event[T, Z]) SubscribeCtx(fn Z, ctx ctxInter, keys ...Key) Sync {
	e.addCallback(-1, fn, ctx, keys...)

	go func() {
		ctx.Wait()
		e.removeCtx(ctx, keys...)
	}()

	return Sync{ctx}
}

func (e *Event[T, Z]) Remove(keys ...Key) {
	e.removeCtx(nil, keys...)
}

func (e *Event[T, Z]) removeCtx(ctx ctxInter, keys ...Key) {
	e.removeCallback(ctx, keys...)
}

func (e *Event[T, Z]) addCallback(idx int, fn Z, ctx ctxInter, keys ...Key) {
	cb := callback[T, Z]{fn, ctx, keys}

	if cb.isNil() {
		panic(fmt.Errorf("Subscribe can not be: %v", fn))
	}

	for {
		oldPtr := e.Callbacks.Load()
		oldSlice := *oldPtr

		if idx == -1 {
			idx = len(oldSlice)
		}

		if idx < 0 || idx > len(oldSlice) {
			panic("subscribe index out of bounds")
		}

		newSlice := make([]callback[T, Z], len(oldSlice)+1)
		copy(newSlice[:idx], oldSlice[:idx])
		newSlice[idx] = cb
		copy(newSlice[idx+1:], oldSlice[idx:])

		if e.Callbacks.CompareAndSwap(oldPtr, &newSlice) {
			return
		}
	}
}

func (e *Event[T, Z]) removeCallback(ctx ctxInter, keys ...Key) {
	if ctx != nil {
		ctx.CancelWithErr(ErrCanceled)
	}

	isMatch := func(cb callback[T, Z]) bool {
		if len(keys) > 0 && slices.Equal(cb.Keys, keys) {
			return true
		} else if cb.Ctx != nil && cb.Ctx == ctx {
			return true
		}
		return false
	}

	for {
		oldPtr := e.Callbacks.Load()
		oldSlice := *oldPtr

		newSlice := slices.DeleteFunc(oldSlice, isMatch)

		if len(newSlice) == len(oldSlice) {
			return
		}

		if e.Callbacks.CompareAndSwap(oldPtr, &newSlice) {
			break
		}
	}
}

func (e *Event[T, Z]) Publish(arg T) {
	slice := *e.Callbacks.Load()
	for i := range slice {
		if slice[i].Alive() {
			slice[i].Callback.Run(arg)
		}
	}
}
