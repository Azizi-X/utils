package debug

import (
	"fmt"
	"sync"
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
	stack := makeStack(err, 4, stackOptions{
		strip:    d.strip,
		maxDepth: d.maxDepth,
		calls:    d.Calls,
	})

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
