package utils

import (
	"cmp"
	"context"
	"errors"
	"time"
)

var ErrContextTimeout = errors.New("timeout")

type Context struct {
	context      context.Context
	cancelFunc   context.CancelFunc
	stopOnCancel func() bool
	reason       string
	err          error
	canceled     bool
}

func (ctx *Context) NewCtx() *Context {
	return NewContext(ctx.context)
}

func (ctx *Context) NewCtxTimeout(timeout time.Duration) *Context {
	return NewCtxTimeout(ctx.context, timeout)
}

func (ctx *Context) Err() error {
	return cmp.Or(ctx.err, ctx.context.Err())
}

func (ctx *Context) Context() context.Context {
	return ctx.context
}

func (ctx *Context) Wait() {
	<-ctx.C()
}

func (ctx *Context) Done() bool {
	return ctx.canceled
}

func (ctx *Context) C() <-chan struct{} {
	return ctx.context.Done()
}

func (ctx *Context) Alive() bool {
	return ctx.context.Err() == nil
}

func (ctx *Context) IsE(err error) bool {
	return errors.Is(ctx.Err(), err)
}

func (ctx *Context) IsR(reason string) bool {
	return ctx.reason == reason
}

func (ctx *Context) SetOnCancel(fn func(error)) {
	if ctx.stopOnCancel != nil {
		ctx.stopOnCancel()
		ctx.stopOnCancel = nil
	}

	if fn != nil {
		ctx.stopOnCancel = context.AfterFunc(ctx.context, func() {
			fn(ctx.Err())
		})
	}
}

func (ctx *Context) CancelWithErr(err error) {
	if err == nil {
		ctx.Cancel()
		return
	}

	if ctx.err == nil {
		ctx.err = err
	}

	ctx.CancelWithReason(ctx.err.Error())
}

func (ctx *Context) CancelWithReason(reason string) {
	if ctx.reason == "" {
		ctx.reason = reason
	}
	if ctx.err == nil {
		ctx.err = errors.New(reason)
	}
	ctx.Cancel()
}

func (ctx *Context) Cancel() {
	ctx.canceled = true
	ctx.cancelFunc()
}

func NewContext(parent context.Context) *Context {
	ctx, cancel := context.WithCancel(parent)

	return &Context{
		context:    ctx,
		cancelFunc: cancel,
	}
}

func NewCtxTimeout(parent context.Context, timeout time.Duration) *Context {
	ctx := NewContext(parent)
	ticker := time.After(timeout)

	go func() {
		select {
		case <-ctx.C():
		case <-ticker:
			ctx.CancelWithErr(ErrContextTimeout)
		}
	}()

	return ctx
}
