package utils

import (
	"slices"
	"sync"
)

type EvictingSet[T comparable] struct {
	values      map[T]struct{}
	max         int
	mu          sync.RWMutex `json:"-"`
	Broadcaster *Broadcaster[T]
}

func (rd *EvictingSet[T]) Add(id T) {
	rd.Broadcaster.Broadcast(id)

	rd.mu.Lock()
	rd.values[id] = struct{}{}

	for len(rd.values) > rd.max {
		for key := range rd.values {
			delete(rd.values, key)
			break
		}
	}
	rd.mu.Unlock()
}

func (rd *EvictingSet[T]) Remove(id T) {
	rd.mu.Lock()
	delete(rd.values, id)
	rd.mu.Unlock()
}

func (rd *EvictingSet[T]) Exists(ids ...T) bool {
	rd.mu.RLock()
	exists := slices.ContainsFunc(ids, func(id T) bool {
		_, ok := rd.values[id]
		return ok
	})
	rd.mu.RUnlock()
	return exists
}

func NewEvictingSet[T comparable](max int) *EvictingSet[T] {
	return &EvictingSet[T]{values: map[T]struct{}{}, max: max, Broadcaster: NewBroadcaster[T]()}
}
