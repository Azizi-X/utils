package utils

import (
	"context"
	"errors"
	"time"
)

type Context struct {
	Context    context.Context    `json:"-"`
	CancelFunc context.CancelFunc `json:"-"`
	onCancel   func(reason string)
	Reason     string
	Err        error
	canceled   bool
}

func (ctx *Context) NewCtx() Context {
	return NewContext(ctx.Context)
}

func (ctx *Context) NewCtxTimeout(timeout time.Duration) Context {
	return NewCtxTimeout(ctx.Context, timeout)
}

func (ctx *Context) GetContextErr() error {
	return ctx.Context.Err()
}

func (ctx *Context) Wait() {
	<-ctx.C()
}

func (ctx *Context) Done() bool {
	return ctx.canceled
}

func (ctx *Context) C() <-chan struct{} {
	return ctx.Context.Done()
}

func (ctx *Context) Alive() bool {
	return ctx.Context.Err() == nil
}

func (ctx *Context) SetOnCancel(onCancel func(reason string)) {
	ctx.onCancel = onCancel
}

func (ctx *Context) CancelWithErr(err error) {
	if ctx.Err == nil {
		ctx.Err = err
	}
	ctx.CancelWithReason(err.Error())
}

func (ctx *Context) CancelWithReason(reason string) {
	if ctx.Reason == "" {
		ctx.Reason = reason
	}
	if ctx.Err == nil {
		ctx.Err = errors.New(reason)
	}
	ctx.Cancel()
}

func (ctx *Context) Cancel() {
	canceled := ctx.canceled
	ctx.canceled = true

	ctx.CancelFunc()

	if !canceled && ctx.onCancel != nil {
		go ctx.onCancel(ctx.Reason)
	}
}

func NewContext(parent context.Context) Context {
	ctx, cancel := context.WithCancel(parent)

	return Context{
		Context:    ctx,
		CancelFunc: cancel,
	}
}

func NewCtxTimeout(parent context.Context, timeout time.Duration) Context {
	ctx := NewContext(parent)
	ticker := time.After(timeout)

	go func() {
		select {
		case <-ctx.C():
		case <-ticker:
			ctx.CancelWithReason("timeout")
		}
	}()

	return ctx
}
