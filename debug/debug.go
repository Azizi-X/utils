package debug

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/mem"
)

const (
	maxDepth = 32
)

type Debugger struct {
	mu       sync.Mutex
	callback []func(err error, stack Stack)
	maxDepth int
	Calls    int
}

type MemStats struct {
	Total       uint64
	Alloc       uint64
	TotalAlloc  uint64
	Mallocs     uint64
	HeapAlloc   uint64
	HeapObjects uint64
}

type Stack struct {
	Error    error
	Time     int64
	Frames   []StackFrame
	MemStats MemStats
}

type StackFrame struct {
	Function string
	File     string
	Line     int
}

func (d *Debugger) frames(skip int) (stack []StackFrame) {
	if d == nil {
		return nil
	}

	pc := make([]uintptr, d.maxDepth)
	n := runtime.Callers(skip, pc)
	frames := runtime.CallersFrames(pc[:n])

	for {
		frame, more := frames.Next()
		if !strings.HasPrefix(frame.Function, "runtime.") {
			stack = append(stack, StackFrame{
				Function: frame.Function,
				File:     frame.File,
				Line:     frame.Line,
			})
		}
		if !more {
			return stack
		}
	}
}

func (d *Debugger) memStats() MemStats {
	if d == nil {
		return MemStats{}
	}

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	memory, _ := mem.VirtualMemory()

	return MemStats{
		Total:       memory.Total,
		Alloc:       stats.Alloc,
		TotalAlloc:  stats.TotalAlloc,
		Mallocs:     stats.Mallocs,
		HeapAlloc:   stats.HeapAlloc,
		HeapObjects: stats.HeapObjects,
	}
}

func (d *Debugger) MakeStack(err error, skip int) Stack {
	frames := d.frames(skip)
	memStats := d.memStats()

	return Stack{
		Error:    err,
		Time:     time.Now().UnixMilli(),
		Frames:   frames,
		MemStats: memStats,
	}
}

func (d *Debugger) Publish(msg any, formats ...any) error {
	if d == nil {
		return nil
	}

	var err error

	switch v := msg.(type) {
	case error:
		err = v
	case string:
		err = fmt.Errorf(v, formats...)
	default:
		err = fmt.Errorf(fmt.Sprint(msg), formats...)
	}

	if err == nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.Calls++
	stack := d.MakeStack(err, 4)

	for _, callback := range d.callback {
		go callback(err, stack)
	}
	return err
}

func (d *Debugger) SetMaxDepth(depth int) *Debugger {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.maxDepth = depth
	return d
}

func (d *Debugger) AddCallback(callback func(err error, stack Stack)) *Debugger {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.callback = append(d.callback, callback)
	return d
}

func NewDebugger() *Debugger {
	return &Debugger{
		maxDepth: maxDepth,
	}
}
