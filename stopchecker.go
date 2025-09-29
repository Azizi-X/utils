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

func (sc *StopChecker) Loop(d time.Duration, fn func()) bool {
	fn()

	for {
		select {
		case <-time.After(d):
			fn()
		case <-sc.Ctx.C():
			return sc.ShouldStop
		}
	}
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
