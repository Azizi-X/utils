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

type mapCore[K comparable, V any] struct {
	mu     sync.RWMutex
	values map[K]V
}

type Map[K comparable, V any] struct {
	rawCore *mapCore[K, V]
}

type mapValues[K comparable, V any] struct {
	Key   K
	Value V
}

type mapValuesList[K comparable, V any] []mapValues[K, V]

func (lst mapValuesList[K, V]) Keys() (keys []K) {
	for _, value := range lst {
		keys = append(keys, value.Key)
	}

	return
}

func (lst mapValuesList[K, V]) Values() (values []V) {
	for _, value := range lst {
		values = append(values, value.Value)
	}

	return
}

func (mp *Map[K, V]) Exists(key K) bool {
	if mp == nil {
		return false
	}

	core := mp.init()

	core.mu.RLock()
	_, exists := core.values[key]
	core.mu.RUnlock()
	return exists
}

func (mp *Map[K, V]) Length() int {
	if mp == nil {
		return 0
	}

	core := mp.init()

	core.mu.RLock()
	length := len(core.values)
	core.mu.RUnlock()
	return length
}

func (mp *Map[K, V]) ContainsFunc(fn func(V) bool) bool {
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

func (mp *Map[K, V]) GetKeys() (keys []K) {
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

func (mp *Map[K, V]) GetList(clear ...bool) (lst []V) {
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

func (mp *Map[K, V]) GetListAndMap() (lst []V, mapList map[K]V) {
	if mp == nil {
		return nil, map[K]V{}
	}

	core := mp.init()

	core.mu.RLock()
	for _, value := range core.values {
		lst = append(lst, value)
	}
	mapList = make(map[K]V, len(core.values))
	maps.Copy(mapList, core.values)
	core.mu.RUnlock()
	return
}

func (mp *Map[K, V]) GetMap(clear ...bool) map[K]V {
	if mp == nil {
		return map[K]V{}
	}

	core := mp.init()

	doClear := len(clear) > 0 && clear[0]

	if doClear {
		core.mu.Lock()
	} else {
		core.mu.RLock()
	}

	copy := make(map[K]V, len(core.values))
	maps.Copy(copy, core.values)

	if doClear {
		core.clearUnsafe()
		core.mu.Unlock()
	} else {
		core.mu.RUnlock()
	}

	return copy
}

func (mp *Map[K, V]) SetMap(value map[K]V) {
	if mp == nil {
		return
	}

	core := mp.init()

	core.mu.Lock()
	core.values = value
	core.mu.Unlock()
}

func (mp *Map[K, V]) Set(key K, value V, conds ...func(length int) (set bool, clear bool)) bool {
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

func (mp *Map[K, V]) SetUniqueFn(key K, fn func(exists bool) (V, bool)) bool {
	if mp == nil {
		return false
	}

	core := mp.init()

	core.mu.Lock()
	_, exists := core.values[key]

	value, ok := fn(exists)

	if !exists && ok {
		core.values[key] = value
	}
	core.mu.Unlock()
	return !exists
}

func (mp *Map[K, V]) SetUnique(key K, value V) bool {
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

func (mp *Map[K, V]) Get(key K) (V, bool) {
	if mp == nil {
		var empty V
		return empty, false
	}

	core := mp.init()

	core.mu.RLock()
	value, exists := core.values[key]
	core.mu.RUnlock()
	return value, exists
}

func (mp *Map[K, V]) GetFrom(keys ...K) (V, bool) {
	if mp == nil {
		var empty V
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

	var empty V
	return empty, false
}

func (mp *Map[K, V]) GetSetFn(key K, fn func() V, conds ...func(V) bool) (V, bool) {
	if mp == nil {
		var empty V
		return empty, false
	}

	core := mp.init()

	core.mu.Lock()
	v, exists := core.values[key]
	if !exists {
		v = fn()
		core.values[key] = v
	}

	for _, cond := range conds {
		if !cond(v) {
			core.mu.Unlock()
			var empty V
			return empty, false
		}
	}

	core.mu.Unlock()
	return v, exists
}

func (mp *Map[K, V]) GetSet(key K, value V) (V, bool) {
	if mp == nil {
		var empty V
		return empty, false
	}

	core := mp.init()

	core.mu.Lock()
	v, exists := core.values[key]
	if !exists {
		v = value
		core.values[key] = v
	}
	core.mu.Unlock()
	return v, exists
}

func (mp *Map[K, V]) DeleteFunc(fn func(key K, value V) bool) mapValuesList[K, V] {
	if mp == nil {
		return nil
	}

	core := mp.init()

	core.mu.Lock()
	var deleted mapValuesList[K, V]

	for key, value := range core.values {
		if fn(key, value) {
			deleted = append(deleted, mapValues[K, V]{
				Key:   key,
				Value: value,
			})
			delete(core.values, key)
		}
	}
	core.mu.Unlock()

	return deleted
}

func (mp *Map[K, V]) Remove(key K) (V, bool) {
	return mp.RemoveCond(key, nil)
}

func (mp *Map[K, V]) RemoveCond(key K, cond func(V) bool) (V, bool) {
	if mp == nil {
		var zero V
		return zero, false
	}

	core := mp.init()

	core.mu.Lock()
	v, exists := core.values[key]
	exists = exists && (cond == nil || cond(v))

	if exists {
		delete(core.values, key)
	}
	core.mu.Unlock()

	return v, exists
}

func (mp *Map[K, V]) Modify(fn func(value *V) bool) mapValuesList[K, V] {
	if mp == nil {
		return nil
	}
	core := mp.init()

	core.mu.Lock()

	var modified mapValuesList[K, V]
	for key := range core.values {
		value := core.values[key]
		if fn(&value) {
			core.values[key] = value
			modified = append(modified, mapValues[K, V]{
				Key:   key,
				Value: value,
			})
		}
	}
	core.mu.Unlock()

	return modified
}

func (core *mapCore[K, V]) clearUnsafe() {
	core.values = map[K]V{}
}

func (mp *Map[K, V]) Clear() {
	if mp == nil {
		return
	}
	core := mp.init()

	core.mu.Lock()
	core.clearUnsafe()
	core.mu.Unlock()
}

func (mp Map[K, V]) MarshalJSON() ([]byte, error) {
	core := mp.init()

	core.mu.RLock()
	defer core.mu.RUnlock()

	return json.Marshal(core.values)
}

func (mp *Map[K, V]) UnmarshalJSON(data []byte) error {
	if mp == nil {
		return nil
	}
	core := mp.init()

	core.mu.Lock()
	defer core.mu.Unlock()

	var values map[K]V
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

func (mp *Map[K, V]) IsZero() bool {
	core := mp.init()

	core.mu.RLock()
	defer core.mu.RUnlock()

	return len(core.values) == 0
}

func (mp *Map[K, V]) init() *mapCore[K, V] {
	addr := (*unsafe.Pointer)(unsafe.Pointer(&mp.rawCore))
	core := atomic.LoadPointer(addr)

	if core == nil {
		newCore := unsafe.Pointer(newMapCore[K, V]())
		atomic.CompareAndSwapPointer(addr, nil, newCore)
		return mp.init()
	}

	return (*mapCore[K, V])(core)
}

func newMapCore[K comparable, V any]() *mapCore[K, V] {
	return &mapCore[K, V]{
		mu:     sync.RWMutex{},
		values: map[K]V{},
	}
}

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		rawCore: newMapCore[K, V](),
	}
}

func NewMapInit[K comparable, V any](values map[K]V) *Map[K, V] {
	newMap := NewMap[K, V]()
	for k, v := range values {
		newMap.Set(k, v)
	}
	return newMap
}
