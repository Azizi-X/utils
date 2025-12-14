package utils

import (
	"context"
	"sync"
	"time"
)

type StopChecker struct {
	Keys       []string
	Ctx        Context
	ShouldStop bool
	once       sync.Once
	processes  *List[*StopChecker]
}

func (sc *StopChecker) LoopC(d time.Duration, fn func(ctx Context)) bool {
	ctx := sc.Ctx.NewCtx()

	fn(ctx)

	for {
		select {
		case <-time.After(d):
			fn(ctx)
		case <-sc.Ctx.C():
			return sc.ShouldStop
		case <-ctx.C():
			return sc.ShouldStop
		}
	}
}

func (sc *StopChecker) Loop(d time.Duration, fn func()) bool {
	return sc.LoopC(d, func(_ Context) {
		fn()
	})
}

func (sc *StopChecker) OnCancel(fn func()) {
	sc.Ctx.SetOnCancel(func(_ string) {
		fn()
	})
}

func (sc *StopChecker) Sleep(t time.Duration) bool {
	select {
	case <-time.After(t):
	case <-sc.Ctx.C():
	}

	return sc.ShouldStop
}

func (sc *StopChecker) Close() {
	sc.once.Do(func() {
		sc.ShouldStop = true
		sc.processes.DeleteFunc(func(value *StopChecker) bool {
			return value.Ctx.Context == sc.Ctx.Context
		})
		sc.Ctx.CancelFunc()
	})
}

func (sc *StopChecker) checkStop() {
	defer sc.Close()
	sc.Ctx.Wait()
}

func NewStopChecker(ctx context.Context, processes *List[*StopChecker], keys ...string) *StopChecker {
	checker := &StopChecker{
		Ctx:       NewContext(ctx),
		processes: processes,
		Keys:      keys,
	}

	processes.Append(checker)

	go checker.checkStop()

	return checker
}




