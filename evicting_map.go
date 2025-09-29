package utils

import (
	"slices"
	"sync"
)

type EvictingMap[K comparable, V any] struct {
	values      map[K]V
	order       []K
	max         int
	mu          sync.RWMutex `json:"-"`
	Broadcaster *Broadcaster[K]
	allowFunc   func(K K, V V) bool
}

func (rd *EvictingMap[K, V]) Add(id K, value V) {
	rd.Broadcaster.Broadcast(id)

	rd.mu.Lock()
	if _, exists := rd.values[id]; !exists {

		if rd.allowFunc != nil && !rd.allowFunc(id, value) {
			rd.mu.Unlock()
			return
		}

		rd.values[id] = value
		rd.order = append(rd.order, id)
	}

	rd.evictItems()

	rd.mu.Unlock()
}

func (rd *EvictingMap[K, V]) Remove(item K) {
	rd.mu.Lock()

	delete(rd.values, item)
	rd.order = slices.DeleteFunc(rd.order, func(k K) bool {
		return k == item
	})

	rd.mu.Unlock()
}

func (rd *EvictingMap[K, V]) Replace(id K, value V) {
	rd.Broadcaster.Broadcast(id)

	rd.mu.Lock()
	_, exists := rd.values[id]

	if rd.allowFunc != nil && !rd.allowFunc(id, value) {
		rd.mu.Unlock()
		return
	}

	if !exists {
		rd.order = append(rd.order, id)
	}

	rd.values[id] = value
	rd.evictItems()

	rd.mu.Unlock()
}

func (rd *EvictingMap[K, V]) Get(item K) (V, bool) {
	rd.mu.RLock()
	value, exists := rd.values[item]
	rd.mu.RUnlock()
	return value, exists
}

func (rd *EvictingMap[K, V]) Clear() {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	rd.values = make(map[K]V)
	rd.order = rd.order[:0]
}

func (rd *EvictingMap[K, V]) Items() []V {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	result := make([]V, 0, len(rd.order))
	for _, key := range rd.order {
		if val, ok := rd.values[key]; ok {
			result = append(result, val)
		}
	}
	return result
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

func (rd *EvictingMap[K, V]) Last() (V, bool) {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	if len(rd.order) == 0 {
		var empty V
		return empty, false
	}

	last := rd.order[len(rd.order)-1]
	value, exists := rd.values[last]

	return value, exists
}

func (rd *EvictingMap[K, V]) AllowFunc(fn func(K K, V V) bool) *EvictingMap[K, V] {
	rd.allowFunc = fn
	return rd
}

func (rd *EvictingMap[K, V]) Len() int {
	rd.mu.RLock()
	length := len(rd.values)
	rd.mu.RUnlock()
	return length
}

func (rd *EvictingMap[K, V]) SetMax(max int) *EvictingMap[K, V] {
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

func (rd *EvictingMap[K, V]) evictItems() {
	for len(rd.values) > rd.max && len(rd.order) > 0 {
		evictKey := rd.order[0]
		rd.order = rd.order[1:]
		delete(rd.values, evictKey)
	}
}

func NewEvictingMap[K comparable, V any](max int) *EvictingMap[K, V] {
	return &EvictingMap[K, V]{values: map[K]V{}, max: max, Broadcaster: NewBroadcaster[K]()}
}
