package utils

import (
	"slices"
	"sync"
)

type Broadcaster[T any] struct {
	subscribers []chan T
	mu          sync.RWMutex `json:"-"`
}

func NewBroadcaster[T any]() *Broadcaster[T] {
	return &Broadcaster[T]{}
}

func (b *Broadcaster[T]) Subscribe() chan T {
	ch := make(chan T, 10)
	b.mu.Lock()
	b.subscribers = append(b.subscribers, ch)
	b.mu.Unlock()
	return ch
}

func (b *Broadcaster[T]) Unsubscribe(ch chan T) {
	b.mu.Lock()
	b.subscribers = slices.DeleteFunc(b.subscribers, func(c chan T) bool {
		if c == ch {
			close(c)
			return true
		}
		return false
	})
	b.mu.Unlock()
}

func (b *Broadcaster[T]) Broadcast(msg T) {
	b.mu.RLock()
	for _, ch := range b.subscribers {
		select {
		case ch <- msg:
		default:
		}
	}
	b.mu.RUnlock()
}
