package EventBus

import (
	"reflect"
	"sync"
)

var Events = New()

type subscriber struct {
	handler reflect.Value
}

type EventBus struct {
	subscribers map[string][]subscriber
	mu          sync.RWMutex
}

func New() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]subscriber),
	}
}

func (b *EventBus) Subscribe(topic string, fn any) {
	v := reflect.ValueOf(fn)
	t := v.Type()
	if t.Kind() != reflect.Func {
		panic("Subscribe requires a function")
	}

	b.mu.Lock()
	b.subscribers[topic] = append(b.subscribers[topic], subscriber{handler: v})
	b.mu.Unlock()
}

func (b *EventBus) Publish(topic string, args ...any) {
	b.mu.RLock()
	subs, ok := b.subscribers[topic]
	b.mu.RUnlock()
	if !ok {
		return
	}

	in := make([]reflect.Value, len(args))
	for i, a := range args {
		in[i] = reflect.ValueOf(a)
	}

	for i := range subs {
		subs[i].handler.Call(in)
	}
}
