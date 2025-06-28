package utils

import (
	"slices"
	"sync"
)

type EvictingMap[K comparable, V any] struct {
	values      map[K]V
	max         int
	mu          sync.RWMutex `json:"-"`
	Broadcaster *Broadcaster[K]
}

func (rd *EvictingMap[K, V]) Add(id K, value V) {
	rd.Broadcaster.Broadcast(id)

	rd.mu.Lock()
	rd.values[id] = value

	for len(rd.values) > rd.max {
		for key := range rd.values {
			delete(rd.values, key)
			break
		}
	}
	rd.mu.Unlock()
}

func (rd *EvictingMap[K, V]) Remove(item K) {
	rd.mu.Lock()
	delete(rd.values, item)
	rd.mu.Unlock()
}

func (rd *EvictingMap[K, V]) Get(item K) (V, bool) {
	rd.mu.RLock()
	value, exists := rd.values[item]
	rd.mu.RUnlock()
	return value, exists
}

func (rd *EvictingMap[K, V]) Has(items ...K) bool {
	rd.mu.RLock()
	exists := slices.ContainsFunc(items, func(item K) bool {
		_, ok := rd.values[item]
		return ok
	})
	rd.mu.RUnlock()
	return exists
}

func NewEvictingMap[K comparable, V any](max int) *EvictingMap[K, V] {
	return &EvictingMap[K, V]{values: map[K]V{}, max: max, Broadcaster: NewBroadcaster[K]()}
}
