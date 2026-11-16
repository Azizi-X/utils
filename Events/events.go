package Events

import (
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
	"unsafe"

	"github.com/Azizi-X/utils"
)

var ErrCanceled = errors.New("context canceled")

type Key string

type eventInter[T any] interface {
	Run(args T)
}

type Event[T any, Z eventInter[T]] struct {
	Name      string
	Callbacks atomic.Pointer[[]*callback[T, Z]]
}

type callback[T any, Z eventInter[T]] struct {
	Callback Z
	Ctx      *utils.Context
	Keys     []Key
}

func New[T any, Z eventInter[T]](name string) *Event[T, Z] {
	event := &Event[T, Z]{
		Name: name,
	}

	initial := make([]*callback[T, Z], 0)
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

func (e *Event[T, Z]) SubscribeCtx(fn Z, ctx *utils.Context, keys ...Key) {
	if ctx == nil {
		panic("Events: nil subscription context")
	}

	e.addCallback(-1, fn, ctx, keys...)

	go func() {
		ctx.Wait()
		e.removeCtx(ctx, keys...)
	}()
}

func (e *Event[T, Z]) Remove(keys ...Key) {
	e.removeCtx(nil, keys...)
}

func (e *Event[T, Z]) removeCtx(ctx *utils.Context, keys ...Key) {
	e.removeCallback(ctx, keys...)
}

func (e *Event[T, Z]) addCallback(idx int, fn Z, ctx *utils.Context, keys ...Key) {
	cb := &callback[T, Z]{fn, ctx, keys}

	if cb.isNil() {
		panic(fmt.Errorf("Subscribe can not be: %v", fn))
	}

	for {
		oldPtr := e.Callbacks.Load()
		oldSlice := *oldPtr

		if idx < 0 || idx > len(oldSlice) {
			idx = len(oldSlice)
		}

		newSlice := make([]*callback[T, Z], len(oldSlice)+1)
		copy(newSlice[:idx], oldSlice[:idx])
		newSlice[idx] = cb
		copy(newSlice[idx+1:], oldSlice[idx:])

		if e.Callbacks.CompareAndSwap(oldPtr, &newSlice) {
			return
		}
	}
}

func (e *Event[T, Z]) removeCallback(ctx *utils.Context, keys ...Key) {
	isMatch := func(cb *callback[T, Z]) bool {
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

		newSlice := slices.DeleteFunc(slices.Clone(oldSlice), isMatch)

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
