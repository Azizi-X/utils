package utils

import (
	"slices"
	"sync"
)

type EvictingSet[T comparable] struct {
	values      map[T]struct{}
	order       []T
	max         int
	mu          sync.RWMutex `json:"-"`
	Broadcaster *Broadcaster[T]
}

func (rd *EvictingSet[T]) Add(id T) {
	rd.Broadcaster.Broadcast(id)

	rd.mu.Lock()

	if _, exists := rd.values[id]; !exists {
		rd.values[id] = struct{}{}
		rd.order = append(rd.order, id)
	}

	for len(rd.values) > rd.max {
		for key := range rd.values {
			delete(rd.values, key)
			break
		}
	}

	rd.mu.Unlock()
}

func (rd *EvictingSet[T]) Remove(item T) {
	rd.mu.Lock()
	delete(rd.values, item)

	rd.order = slices.DeleteFunc(rd.order, func(v T) bool {
		return v == item
	})

	rd.mu.Unlock()
}

func (rd *EvictingSet[T]) Exists(items ...T) bool {
	rd.mu.RLock()
	exists := slices.ContainsFunc(items, func(item T) bool {
		_, ok := rd.values[item]
		return ok
	})
	rd.mu.RUnlock()
	return exists
}

func (rd *EvictingSet[T]) Len() int {
	rd.mu.RLock()
	length := len(rd.values)
	rd.mu.RUnlock()
	return length
}

func NewEvictingSet[T comparable](max int) *EvictingSet[T] {
	return &EvictingSet[T]{values: map[T]struct{}{}, max: max, Broadcaster: NewBroadcaster[T]()}
}
