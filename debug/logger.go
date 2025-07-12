package debug

import (
	"errors"
	"fmt"
	"sync"
)

type Logger struct {
	mu       sync.Mutex
	callback []func(message string, stack Stack)
	maxDepth int
	strip    bool
	Calls    int
}

func (l *Logger) SetStrip(strip bool) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.strip = strip
	return l
}

func (l *Logger) SetMaxDepth(depth int) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.maxDepth = depth
	return l
}

func (l *Logger) Verbose(msg string, formats ...any) {
	if l == nil {
		return
	}

	msg = fmt.Sprintf(msg, formats...)

	l.mu.Lock()
	defer l.mu.Unlock()

	l.Calls++

	stack := makeStack(errors.New(msg), 4, stackOptions{
		strip:    l.strip,
		maxDepth: l.maxDepth,
		calls:    l.Calls,
	})

	for _, callback := range l.callback {
		go callback(msg, stack)
	}
}

func (l *Logger) AddCallback(callback func(message string, stack Stack)) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.callback = append(l.callback, callback)
	return l
}

func NewLogger() *Logger {
	return &Logger{
		maxDepth: maxDepth,
	}
}
