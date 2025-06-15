package EventBus

import (
	"reflect"
	"sync"
	"unsafe"
)

type TypeEventBus struct {
	subscribers map[string][]subscriber
	mu          sync.RWMutex
}

func NewTypeBus() *TypeEventBus {
	return &TypeEventBus{
		subscribers: make(map[string][]subscriber),
	}
}

func keyFromTypes(types []reflect.Type) string {
	var buf []byte

	for i, t := range types {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = append(buf, t.PkgPath()...)
		buf = append(buf, '.')
		buf = append(buf, t.Name()...)
	}

	return *(*string)(unsafe.Pointer(&buf))
}

func (b *TypeEventBus) Subscribe(fn any) {
	v := reflect.ValueOf(fn)
	t := v.Type()
	if t.Kind() != reflect.Func {
		panic("Subscribe requires a function")
	}

	numIn := t.NumIn()
	argTypes := make([]reflect.Type, numIn)
	for i := range numIn {
		argTypes[i] = t.In(i)
	}
	key := keyFromTypes(argTypes)

	b.mu.Lock()
	b.subscribers[key] = append(b.subscribers[key], subscriber{handler: v})
	b.mu.Unlock()
}

func (b *TypeEventBus) Publish(args ...any) {
	argTypes := make([]reflect.Type, len(args))
	for i, a := range args {
		argTypes[i] = reflect.TypeOf(a)
	}
	key := keyFromTypes(argTypes)

	b.mu.RLock()
	subs, ok := b.subscribers[key]
	b.mu.RUnlock()
	if !ok {
		return
	}

	in := make([]reflect.Value, len(args))
	for i, a := range args {
		in[i] = reflect.ValueOf(a)
	}

	for _, sub := range subs {
		sub.handler.Call(in)
	}
}
