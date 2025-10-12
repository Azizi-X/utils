package utils

import (
	"encoding/json"
	"fmt"
	"maps"
	"sync"
)

var (
	IgnoreInvalidMapTypes = true
)

type Map[T any] struct {
	mu     *sync.RWMutex
	values map[string]T
}

type mapValues[T any] struct {
	Key   string
	Value T
}

type mapValuesList[T any] []mapValues[T]

func (lst mapValuesList[T]) Keys() (keys []string) {
	for _, value := range lst {
		keys = append(keys, value.Key)
	}

	return
}

func (lst mapValuesList[T]) Values() (values []T) {
	for _, value := range lst {
		values = append(values, value.Value)
	}

	return
}

func (mp *Map[T]) Exists(key string) bool {
	if mp == nil {
		return false
	}

	mp.checkMu()

	mp.mu.RLock()
	_, exists := mp.values[key]
	mp.mu.RUnlock()
	return exists
}

func (mp *Map[T]) Length() int {
	if mp == nil {
		return 0
	}

	mp.checkMu()

	mp.mu.RLock()
	length := len(mp.values)
	mp.mu.RUnlock()
	return length
}

func (mp *Map[T]) GetMapList(clear ...bool) (lst []T) {
	if mp == nil {
		return []T{}
	}

	mp.checkMu()

	doClear := len(clear) > 0 && clear[0]

	if doClear {
		mp.mu.Lock()
	} else {
		mp.mu.RLock()
	}

	for _, value := range mp.values {
		lst = append(lst, value)
	}

	if doClear {
		mp.clearUnsafe()
		mp.mu.Unlock()
	} else {
		mp.mu.RUnlock()
	}
	return
}

func (mp *Map[T]) GetListAndMap() (lst []T, mapList map[string]T) {
	if mp == nil {
		return []T{}, map[string]T{}
	}

	mp.checkMu()

	mp.mu.RLock()
	for _, value := range mp.values {
		lst = append(lst, value)
	}
	mapList = make(map[string]T, len(mp.values))
	maps.Copy(mapList, mp.values)
	mp.mu.RUnlock()
	return
}

func (mp *Map[T]) GetMap(clear ...bool) map[string]T {
	if mp == nil {
		return map[string]T{}
	}

	mp.checkMu()

	doClear := len(clear) > 0 && clear[0]

	if doClear {
		mp.mu.Lock()
	} else {
		mp.mu.RLock()
	}

	copy := make(map[string]T, len(mp.values))
	maps.Copy(copy, mp.values)

	if doClear {
		mp.clearUnsafe()
		mp.mu.Unlock()
	} else {
		mp.mu.RUnlock()
	}

	return copy
}

func (mp *Map[T]) SetMap(value map[string]T) {
	if mp == nil {
		return
	}

	mp.checkMu()

	mp.mu.Lock()
	mp.values = value
	mp.mu.Unlock()
}

func (mp *Map[T]) Set(key string, value T, conds ...func(length int) (set bool, clear bool)) bool {
	if mp == nil {
		return false
	}

	mp.checkMu()

	mp.mu.Lock()

	_, exists := mp.values[key]

	for _, cond := range conds {
		if set, clear := cond(len(mp.values)); !set {
			if clear {
				mp.clearUnsafe()
			}
			mp.mu.Unlock()
			return exists
		}
	}

	mp.values[key] = value
	mp.mu.Unlock()
	return exists
}

func (mp *Map[T]) SetUnique(key string, value T) bool {
	if mp == nil {
		return false
	}

	mp.checkMu()

	mp.mu.Lock()
	_, exists := mp.values[key]
	if !exists {
		mp.values[key] = value
	}
	mp.mu.Unlock()
	return !exists
}

func (mp *Map[T]) Get(key string) (T, bool) {
	if mp == nil {
		var empty T
		return empty, false
	}

	mp.checkMu()

	mp.mu.RLock()
	value, exists := mp.values[key]
	mp.mu.RUnlock()
	return value, exists
}

func (mp *Map[T]) GetFrom(keys ...string) (T, bool) {
	if mp == nil {
		var empty T
		return empty, false
	}

	mp.checkMu()

	mp.mu.RLock()
	defer mp.mu.RUnlock()

	for _, key := range keys {
		value, exists := mp.values[key]
		if exists {
			return value, exists
		}
	}

	var empty T
	return empty, false
}

func (mp *Map[T]) GetSet(key string, value T) (T, bool) {
	if mp == nil {
		var empty T
		return empty, false
	}

	mp.checkMu()

	mp.mu.Lock()
	_, exists := mp.values[key]
	if !exists {
		mp.values[key] = value
	}
	v := mp.values[key]
	mp.mu.Unlock()
	return v, exists
}

func (mp *Map[T]) DeleteFunc(fn func(key string, value T) bool) mapValuesList[T] {
	if mp == nil {
		return nil
	}

	mp.checkMu()

	mp.mu.Lock()
	var deleted mapValuesList[T]

	for key, value := range mp.values {
		if fn(key, value) {
			deleted = append(deleted, mapValues[T]{
				Key:   key,
				Value: value,
			})
			delete(mp.values, key)
		}
	}
	mp.mu.Unlock()

	return deleted
}

func (mp *Map[T]) Remove(key string) *T {
	if mp == nil {
		return nil
	}

	mp.checkMu()

	mp.mu.Lock()
	v, exists := mp.values[key]
	delete(mp.values, key)
	mp.mu.Unlock()

	if exists {
		return &v
	}

	return nil
}

func (mp *Map[T]) Modify(fn func(mp *map[string]T)) {
	if mp == nil {
		return
	}
	mp.checkMu()

	mp.mu.Lock()
	fn(&mp.values)
	mp.mu.Unlock()
}

func (mp *Map[T]) clearUnsafe() {
	mp.values = map[string]T{}
}

func (mp *Map[T]) Clear() {
	if mp == nil {
		return
	}
	mp.checkMu()

	mp.mu.Lock()
	mp.clearUnsafe()
	mp.mu.Unlock()
}

func (mp Map[T]) MarshalJSON() ([]byte, error) {
	mp.checkMu()

	mp.mu.Lock()
	defer mp.mu.Unlock()

	return json.Marshal(mp.values)
}

func (mp *Map[T]) UnmarshalJSON(data []byte) error {
	if mp == nil {
		return nil
	}
	mp.checkMu()

	mp.mu.Lock()
	defer mp.mu.Unlock()

	var values map[string]T
	if err := json.Unmarshal(data, &values); err != nil {
		if _, ok := err.(*json.UnmarshalTypeError); ok && IgnoreInvalidMapTypes {
			fmt.Println("[MAP] Ignoring invalid:", err)
			return nil
		}

		return err
	}

	mp.values = values
	return nil
}

func (mp *Map[T]) IsZero() bool {
	mp.checkMu()

	mp.mu.RLock()
	defer mp.mu.RUnlock()

	return len(mp.values) == 0
}

func (mp *Map[T]) checkMu() {
	if mp.mu == nil || mp.values == nil {
		checkMu.Lock()
		if mp.mu == nil {
			mp.mu = &sync.RWMutex{}
		}

		if mp.values == nil {
			mp.mu.Lock()
			mp.values = map[string]T{}
			mp.mu.Unlock()
		}
		checkMu.Unlock()
	}
}

func NewMap[T any]() *Map[T] {
	return &Map[T]{
		values: map[string]T{},
		mu:     &sync.RWMutex{},
	}
}

func NewMapInit[T any](values map[string]T) *Map[T] {
	newMap := NewMap[T]()
	for k, v := range values {
		newMap.Set(k, v)
	}
	return newMap
}

