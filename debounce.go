package utils

import (
	"context"
	"sync"
	"time"
)

type Debouncer struct {
	ctx   context.Context
	mu    sync.Mutex
	after time.Duration
	timer *time.Timer
	cache Map[int64, *Debouncer]
}

func NewDebouncer(ctx context.Context, after time.Duration) *Debouncer {
	return &Debouncer{ctx: ctx, after: after}
}

func (d *Debouncer) Cancel() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.unsafeCancel()
}

func (d *Debouncer) CancelFor(key int64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if debouncer, ok := d.cache.Remove(key); ok {
		debouncer.Cancel()
	}
}

func (d *Debouncer) unsafeCancel() {
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
}

func (d *Debouncer) Do(f func(), direct ...bool) {
	if len(direct) > 0 && direct[0] {
		d.Cancel()
		f()
		return
	}

	d.add(f)
}

func (d *Debouncer) DoFor(key int64, f func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	debouncer, _ := d.cache.GetSetFn(key, func() *Debouncer {
		return NewDebouncer(d.ctx, d.after)
	})
	debouncer.Do(f)
}

func (d *Debouncer) add(f func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.unsafeCancel()

	if d.ctx.Err() != nil {
		return
	}

	d.timer = time.AfterFunc(d.after, func() {
		if d.ctx.Err() == nil {
			f()
		}
	})
}
