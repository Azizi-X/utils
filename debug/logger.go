package debug

import (
	"errors"
	"fmt"
	"sync"
)

const (
	Verbose = 1
)

type Logger struct {
	mu       sync.Mutex
	callback []func(message string, stack Stack, level int)
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

func (l *Logger) Verbose(msg string, formats ...any) error {
	return l.Log(Verbose, msg, formats...)
}

func (l *Logger) Log(level int, msg string, formats ...any) error {
	if l == nil {
		return nil
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
		go callback(msg, stack, level)
	}

	return errors.New(msg)
}

func (l *Logger) AddCallback(callback func(message string, stack Stack, level int)) *Logger {
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
