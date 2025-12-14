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
	allowFunc   func(T T) bool
}

func (rd *EvictingSet[T]) Add(id T) {
	if rd.Broadcaster != nil {
		rd.Broadcaster.Broadcast(id)
	}

	rd.mu.Lock()

	if _, exists := rd.values[id]; !exists {
		if rd.allowFunc != nil && !rd.allowFunc(id) {
			rd.mu.Unlock()
			return
		}

		rd.values[id] = struct{}{}
		rd.order = append(rd.order, id)
	}

	rd.evictItems()

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

func (rd *EvictingSet[T]) Items() []T {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	result := make([]T, 0, len(rd.order))
	for _, key := range rd.order {
		if _, ok := rd.values[key]; ok {
			result = append(result, key)
		}
	}
	return result
}

func (rd *EvictingSet[T]) Clear() {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	rd.values = make(map[T]struct{})
	rd.order = rd.order[:0]
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

func (rd *EvictingSet[T]) AllowFunc(fn func(T T) bool) *EvictingSet[T] {
	rd.allowFunc = fn
	return rd
}

func (rd *EvictingSet[T]) Len() int {
	rd.mu.RLock()
	length := len(rd.values)
	rd.mu.RUnlock()
	return length
}

func (rd *EvictingSet[T]) SetMax(max int) *EvictingSet[T] {
	rd.mu.Lock()
	if rd.max == max {
		rd.mu.Unlock()
		return rd
	}

	rd.max = max
	rd.evictItems()
	rd.mu.Unlock()

	return rd
}

func (rd *EvictingSet[T]) evictItems() {
	for len(rd.values) > rd.max && len(rd.order) > 0 {
		evictKey := rd.order[0]
		rd.order = rd.order[1:]
		delete(rd.values, evictKey)
	}
}

func (d *EvictingSet[T]) WithBroadcaster() *EvictingSet[T] {
	d.Broadcaster = NewBroadcaster[T]()
	return d
}

func NewEvictingSet[T comparable](max int) *EvictingSet[T] {
	return &EvictingSet[T]{values: map[T]struct{}{}, max: max}
}
