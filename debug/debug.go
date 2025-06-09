package debug

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
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
	callback []func(err error, stack Stack) error
	maxDepth int
	strip    bool
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
	Error    string
	Time     int64
	Frames   []StackFrame
	MemStats MemStats
	Total    int
}

type StackFrame struct {
	Function string
	File     string
	Line     int
}

func (s *Stack) Hash(extra ...string) string {
	var builder strings.Builder

	builder.WriteString(s.Error)

	for i := range extra {
		builder.WriteString(extra[i])
	}

	for _, frame := range s.Frames {
		builder.WriteString(frame.Function)
		builder.WriteString(frame.File)
		builder.WriteString(string(rune(frame.Line)))
	}

	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
}

func stripPath(frameFile string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return frameFile
	}

	rel, err := filepath.Rel(cwd, frameFile)
	if err != nil {
		return frameFile
	}

	return filepath.ToSlash(rel)
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
			var file string

			if d.strip {
				file = stripPath(frame.File)
			} else {
				file = frame.File
			}

			stack = append(stack, StackFrame{
				Function: frame.Function,
				File:     file,
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

	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	return Stack{
		Error:    errStr,
		Time:     time.Now().UnixMilli(),
		Frames:   frames,
		MemStats: memStats,
		Total:    d.Calls,
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

func (d *Debugger) SetStrip(strip bool) *Debugger {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.strip = strip
	return d
}

func (d *Debugger) SetMaxDepth(depth int) *Debugger {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.maxDepth = depth
	return d
}

func (d *Debugger) AddCallback(callback func(err error, stack Stack) error) *Debugger {
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
