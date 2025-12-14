package debug

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/mem"
)

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

func frames(skip int, options stackOptions) (stack []StackFrame) {
	pc := make([]uintptr, options.maxDepth)
	n := runtime.Callers(skip, pc)
	frames := runtime.CallersFrames(pc[:n])

	for {
		frame, more := frames.Next()
		if !strings.HasPrefix(frame.Function, "runtime.") {
			var file string

			if options.strip {
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

func memStats() MemStats {
	var total uint64
	var stats runtime.MemStats

	runtime.ReadMemStats(&stats)

	if memory, _ := mem.VirtualMemory(); memory != nil {
		total = memory.Total
	}

	return MemStats{
		Total:       total,
		Alloc:       stats.Alloc,
		TotalAlloc:  stats.TotalAlloc,
		Mallocs:     stats.Mallocs,
		HeapAlloc:   stats.HeapAlloc,
		HeapObjects: stats.HeapObjects,
	}
}

func makeStack(err error, skip int, options stackOptions) Stack {
	frames := frames(skip, options)
	memStats := memStats()

	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	return Stack{
		Error:    errStr,
		Time:     time.Now().UnixMilli(),
		Frames:   frames,
		MemStats: memStats,
		Total:    options.calls,
	}
}
