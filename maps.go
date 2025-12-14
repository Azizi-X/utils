package utils

import (
	"encoding/json"
	"fmt"
	"maps"
	"sync"
	"sync/atomic"
	"unsafe"
)

var (
	IgnoreInvalidMapTypes = true
)

type mapCore[T any] struct {
	mu     sync.RWMutex
	values map[string]T
}

type Map[T any] struct {
	rawCore *mapCore[T]
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

	core := mp.init()

	core.mu.RLock()
	_, exists := core.values[key]
	core.mu.RUnlock()
	return exists
}

func (mp *Map[T]) Length() int {
	if mp == nil {
		return 0
	}

	core := mp.init()

	core.mu.RLock()
	length := len(core.values)
	core.mu.RUnlock()
	return length
}

func (mp *Map[T]) ContainsFunc(fn func(T) bool) bool {
	if mp == nil {
		return false
	}

	core := mp.init()

	core.mu.RLock()
	for _, value := range core.values {
		if fn(value) {
			core.mu.RUnlock()
			return true
		}
	}
	core.mu.RUnlock()
	return false
}

func (mp *Map[T]) GetKeys() (keys []string) {
	if mp == nil {
		return nil
	}

	core := mp.init()

	core.mu.RLock()

	for key := range core.values {
		keys = append(keys, key)
	}

	core.mu.RUnlock()
	return
}

func (mp *Map[T]) GetList(clear ...bool) (lst []T) {
	if mp == nil {
		return nil
	}

	core := mp.init()

	doClear := len(clear) > 0 && clear[0]

	if doClear {
		core.mu.Lock()
	} else {
		core.mu.RLock()
	}

	for _, value := range core.values {
		lst = append(lst, value)
	}

	if doClear {
		core.clearUnsafe()
		core.mu.Unlock()
	} else {
		core.mu.RUnlock()
	}
	return
}

func (mp *Map[T]) GetListAndMap() (lst []T, mapList map[string]T) {
	if mp == nil {
		return nil, map[string]T{}
	}

	core := mp.init()

	core.mu.RLock()
	for _, value := range core.values {
		lst = append(lst, value)
	}
	mapList = make(map[string]T, len(core.values))
	maps.Copy(mapList, core.values)
	core.mu.RUnlock()
	return
}

func (mp *Map[T]) GetMap(clear ...bool) map[string]T {
	if mp == nil {
		return map[string]T{}
	}

	core := mp.init()

	doClear := len(clear) > 0 && clear[0]

	if doClear {
		core.mu.Lock()
	} else {
		core.mu.RLock()
	}

	copy := make(map[string]T, len(core.values))
	maps.Copy(copy, core.values)

	if doClear {
		core.clearUnsafe()
		core.mu.Unlock()
	} else {
		core.mu.RUnlock()
	}

	return copy
}

func (mp *Map[T]) SetMap(value map[string]T) {
	if mp == nil {
		return
	}

	core := mp.init()

	core.mu.Lock()
	core.values = value
	core.mu.Unlock()
}

func (mp *Map[T]) Set(key string, value T, conds ...func(length int) (set bool, clear bool)) bool {
	if mp == nil {
		return false
	}

	core := mp.init()

	core.mu.Lock()

	_, exists := core.values[key]

	for _, cond := range conds {
		if set, clear := cond(len(core.values)); !set {
			if clear {
				core.clearUnsafe()
			}
			core.mu.Unlock()
			return exists
		}
	}

	core.values[key] = value
	core.mu.Unlock()
	return exists
}

func (mp *Map[T]) SetUnique(key string, value T) bool {
	if mp == nil {
		return false
	}

	core := mp.init()

	core.mu.Lock()
	_, exists := core.values[key]
	if !exists {
		core.values[key] = value
	}
	core.mu.Unlock()
	return !exists
}

func (mp *Map[T]) Get(key string) (T, bool) {
	if mp == nil {
		var empty T
		return empty, false
	}

	core := mp.init()

	core.mu.RLock()
	value, exists := core.values[key]
	core.mu.RUnlock()
	return value, exists
}

func (mp *Map[T]) GetFrom(keys ...string) (T, bool) {
	if mp == nil {
		var empty T
		return empty, false
	}

	core := mp.init()

	core.mu.RLock()
	defer core.mu.RUnlock()

	for _, key := range keys {
		value, exists := core.values[key]
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

	core := mp.init()

	core.mu.Lock()
	_, exists := core.values[key]
	if !exists {
		core.values[key] = value
	}
	v := core.values[key]
	core.mu.Unlock()
	return v, exists
}

func (mp *Map[T]) DeleteFunc(fn func(key string, value T) bool) mapValuesList[T] {
	if mp == nil {
		return nil
	}

	core := mp.init()

	core.mu.Lock()
	var deleted mapValuesList[T]

	for key, value := range core.values {
		if fn(key, value) {
			deleted = append(deleted, mapValues[T]{
				Key:   key,
				Value: value,
			})
			delete(core.values, key)
		}
	}
	core.mu.Unlock()

	return deleted
}

func (mp *Map[T]) Remove(key string) *T {
	if mp == nil {
		return nil
	}

	core := mp.init()

	core.mu.Lock()
	v, exists := core.values[key]
	delete(core.values, key)
	core.mu.Unlock()

	if exists {
		return &v
	}

	return nil
}

func (mp *Map[T]) Modify(fn func(mp *map[string]T)) {
	if mp == nil {
		return
	}
	core := mp.init()

	core.mu.Lock()
	fn(&core.values)
	core.mu.Unlock()
}

func (core *mapCore[T]) clearUnsafe() {
	core.values = map[string]T{}
}

func (mp *Map[T]) Clear() {
	if mp == nil {
		return
	}
	core := mp.init()

	core.mu.Lock()
	core.clearUnsafe()
	core.mu.Unlock()
}

func (mp Map[T]) MarshalJSON() ([]byte, error) {
	core := mp.init()

	core.mu.RLock()
	defer core.mu.RUnlock()

	return json.Marshal(core.values)
}

func (mp *Map[T]) UnmarshalJSON(data []byte) error {
	if mp == nil {
		return nil
	}
	core := mp.init()

	core.mu.Lock()
	defer core.mu.Unlock()

	var values map[string]T
	if err := json.Unmarshal(data, &values); err != nil {
		if _, ok := err.(*json.UnmarshalTypeError); ok && IgnoreInvalidMapTypes {
			fmt.Println("[MAP] Ignoring invalid:", err)
			return nil
		}

		return err
	}

	core.values = values
	return nil
}

func (mp *Map[T]) IsZero() bool {
	core := mp.init()

	core.mu.RLock()
	defer core.mu.RUnlock()

	return len(core.values) == 0
}

func (mp *Map[T]) init() *mapCore[T] {
	addr := (*unsafe.Pointer)(unsafe.Pointer(&mp.rawCore))
	core := atomic.LoadPointer(addr)

	if core == nil {
		newCore := unsafe.Pointer(newMapCore[T]())
		atomic.CompareAndSwapPointer(addr, nil, newCore)
		return mp.init()
	}

	return (*mapCore[T])(core)
}

func newMapCore[T any]() *mapCore[T] {
	return &mapCore[T]{
		mu:     sync.RWMutex{},
		values: map[string]T{},
	}
}

func NewMap[T any]() *Map[T] {
	return &Map[T]{
		rawCore: newMapCore[T](),
	}
}

func NewMapInit[T any](values map[string]T) *Map[T] {
	newMap := NewMap[T]()
	for k, v := range values {
		newMap.Set(k, v)
	}
	return newMap
}

