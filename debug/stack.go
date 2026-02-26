package debug

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

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
	Total    int
}

type StackFrame struct {
	Function string
	File     string
	Line     int
}

type stackOptions struct {
	strip    bool
	maxDepth int
	calls    int
}

func (s *Stack) Hash(extra ...string) string {
	var builder strings.Builder

	builder.WriteString(s.Error.Error())

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
