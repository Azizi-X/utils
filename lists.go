package utils

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"slices"
	"sort"
	"sync"
)

var (
	checkMu                sync.Mutex
	IgnoreInvalidListTypes = true
)

type List[T any] struct {
	mu     *sync.RWMutex
	limit  *int
	equal  func(a, b T) bool
	values []T
}

func (lst *List[T]) Reverse() {
	if lst == nil {
		return
	}

	lst.checkMu()

	lst.mu.Lock()
	slices.Reverse(lst.values)
	lst.mu.Unlock()
}

func (lst *List[T]) RandomItem() *T {
	if lst == nil {
		return nil
	}

	lst.checkMu()

	lst.mu.RLock()
	randomItem := GetRandomItem(lst.values)
	lst.mu.RUnlock()
	return randomItem
}

func (lst *List[T]) checkLimit() {
	if lst.limit == nil {
		return
	}

	for len(lst.values) > *lst.limit {
		lst.values = lst.values[1:]
	}
}

func (lst *List[T]) SetEqual(fn func(a, b T) bool) *List[T] {
	if lst == nil {
		return nil
	}

	lst.checkMu()

	lst.mu.Lock()
	lst.equal = fn
	lst.mu.Unlock()

	return lst
}

func (lst *List[T]) SetLimit(limit int) *List[T] {
	if lst == nil {
		return nil
	}

	lst.checkMu()

	lst.mu.Lock()
	lst.limit = &limit
	lst.checkLimit()
	lst.mu.Unlock()

	return lst
}

func (lst *List[T]) Contains(value T) bool {
	if lst == nil {
		return false
	}

	lst.checkMu()

	lst.mu.RLock()

	for i := range lst.values {
		if reflect.DeepEqual(lst.values[i], value) {
			lst.mu.RUnlock()
			return true
		}
	}

	lst.mu.RUnlock()
	return false
}

func (lst *List[T]) Last() (T, bool) {
	if lst == nil {
		var empty T
		return empty, false
	}

	lst.checkMu()

	lst.mu.RLock()

	if len(lst.values) > 0 {
		lst.mu.RUnlock()
		return lst.values[len(lst.values)-1], true
	}

	lst.mu.RUnlock()

	var empty T
	return empty, false
}

func (lst *List[T]) GetIndex(index int) (T, bool) {
	if lst != nil {
		lst.checkMu()
		lst.mu.RLock()
		if index >= 0 && index < len(lst.values) {
			lst.mu.RUnlock()
			return lst.values[index], true
		}
		lst.mu.RUnlock()
	}
	var empty T
	return empty, false
}

func (lst *List[T]) Empty() bool {
	return lst.Length() == 0
}

func (lst *List[T]) Length() int {
	if lst == nil {
		return 0
	}

	lst.checkMu()

	lst.mu.RLock()
	length := len(lst.values)
	lst.mu.RUnlock()
	return length
}

func (lst *List[T]) Sort(fn func(i, j T) bool) {
	if lst == nil {
		return
	}

	lst.checkMu()

	lst.mu.Lock()
	sort.Slice(lst.values, func(i, j int) bool {
		return fn(lst.values[i], lst.values[j])
	})
	lst.mu.Unlock()
}

func (lst *List[T]) GetListClear() (values []T) {
	if lst == nil {
		return []T{}
	}

	lst.checkMu()
	lst.mu.Lock()
	values = append(values, lst.values...)
	lst.values = []T{}
	lst.mu.Unlock()
	return
}

func (lst *List[T]) GetList() (values []T) {
	if lst == nil {
		return []T{}
	}

	lst.checkMu()
	lst.mu.RLock()
	values = append(values, lst.values...)
	lst.mu.RUnlock()
	return
}

func (lst *List[T]) Collect(fn func(T) bool) (values []T) {
	if lst == nil {
		return []T{}
	}

	lst.checkMu()
	lst.mu.RLock()

	for i := range lst.values {
		if fn(lst.values[i]) {
			values = append(values, lst.values[i])
		}
	}

	lst.mu.RUnlock()
	return
}

func (lst *List[T]) AppendFunc(value T, fn func(item T) (exists bool)) (added bool) {
	if lst == nil {
		return false
	}
	lst.checkMu()
	lst.mu.Lock()

	if slices.ContainsFunc(lst.values, fn) {
		lst.mu.Unlock()
		return false
	}

	lst.values = append(lst.values, value)
	lst.checkLimit()
	lst.mu.Unlock()
	return true
}

func (lst *List[T]) appendDeepEqual(value T) bool {
	if lst == nil {
		return false
	}

	lst.mu.Lock()
	for i := range lst.values {
		if reflect.DeepEqual(lst.values[i], value) {
			lst.mu.Unlock()
			return false
		}
	}

	lst.values = append(lst.values, value)
	lst.checkLimit()
	lst.mu.Unlock()

	return true
}

func (lst *List[T]) AppendUnique(value T) bool {
	if lst == nil {
		return false
	}

	lst.checkMu()

	if lst.equal == nil {
		return lst.appendDeepEqual(value)
	}

	lst.mu.Lock()

	for i := range lst.values {
		if lst.equal(lst.values[i], value) {
			lst.mu.Unlock()
			return false
		}
	}

	lst.values = append(lst.values, value)
	lst.checkLimit()

	lst.mu.Unlock()
	return true
}

func (lst *List[T]) ContainsFunc(fn func(value T) bool) bool {
	if lst == nil {
		return false
	}

	lst.checkMu()

	lst.mu.RLock()
	defer lst.mu.RUnlock()

	return slices.ContainsFunc(lst.values, fn)
}

func (lst *List[T]) DeleteFuncList(fn func(value T) bool) []T {
	if lst == nil {
		return nil
	}

	lst.checkMu()

	lst.mu.Lock()
	defer lst.mu.Unlock()

	var removed []T

	remove := func(value T) bool {
		remove := fn(value)

		if remove {
			removed = append(removed, value)
		}

		return remove
	}

	lst.values = slices.DeleteFunc(lst.values, remove)

	return removed
}

func (lst *List[T]) DeleteFunc(fn func(value T) bool) bool {
	if lst == nil {
		return false
	}

	lst.checkMu()

	lst.mu.Lock()
	defer lst.mu.Unlock()

	before := len(lst.values)
	lst.values = slices.DeleteFunc(lst.values, fn)

	return before != len(lst.values)
}

func (lst *List[T]) Remove(value T) []T {
	if lst == nil {
		return []T{}
	}

	return lst.DeleteFuncList(func(value T) bool {
		return reflect.DeepEqual(value, value)
	})
}

func (lst *List[T]) Modify(fn func(value *T) bool) []T {
	if lst == nil {
		return nil
	}

	lst.checkMu()

	lst.mu.Lock()

	var modified []T
	for i := range lst.values {
		if fn(&lst.values[i]) {
			modified = append(modified, lst.values[i])
		}
	}

	lst.checkLimit()
	lst.mu.Unlock()

	return modified
}

func (lst *List[T]) Insert(index int, values ...T) {
	if lst == nil {
		return
	}

	lst.checkMu()

	lst.mu.Lock()

	index = max(index, 0)
	index = min(index, len(lst.values))

	lst.values = append(lst.values, make([]T, len(values))...)
	copy(lst.values[index+len(values):], lst.values[index:])
	copy(lst.values[index:], values)

	lst.checkLimit()
	lst.mu.Unlock()
}

func (lst *List[T]) Append(values ...T) {
	if lst == nil {
		return
	}

	lst.checkMu()

	lst.mu.Lock()
	lst.values = append(lst.values, values...)
	lst.checkLimit()
	lst.mu.Unlock()
}

func (lst *List[T]) SetList(values []T) {
	if lst == nil {
		return
	}
	lst.checkMu()

	lst.mu.Lock()
	lst.values = values
	lst.checkLimit()
	lst.mu.Unlock()
}

func (lst *List[T]) Clear() {
	if lst == nil {
		return
	}
	lst.checkMu()

	lst.mu.Lock()
	lst.values = []T{}
	lst.mu.Unlock()
}

func (lst *List[T]) IsZero() bool {
	lst.checkMu()

	lst.mu.RLock()
	defer lst.mu.RUnlock()

	return len(lst.values) == 0
}

func (lst List[T]) MarshalJSON() ([]byte, error) {
	lst.checkMu()

	lst.mu.Lock()
	defer lst.mu.Unlock()

	return json.Marshal(lst.values)
}

func (lst *List[T]) UnmarshalJSON(data []byte) error {
	if lst == nil {
		return nil
	}

	lst.checkMu()

	lst.mu.Lock()
	defer lst.mu.Unlock()

	var values []T
	if err := json.Unmarshal(data, &values); err != nil {
		if _, ok := err.(*json.UnmarshalTypeError); ok && IgnoreInvalidListTypes {
			fmt.Println("[LIST] Ignoring invalid:", err)
			return nil
		}

		return err
	}

	lst.values = values
	return nil
}

func (lst *List[T]) checkMu() {
	if lst.mu == nil {
		checkMu.Lock()
		if lst.mu == nil {
			lst.mu = &sync.RWMutex{}
		}
		checkMu.Unlock()
	}
}

func GetRandomItem[T any](lst []T) *T {
	if len(lst) == 0 {
		return nil
	}

	return &lst[rand.Intn(len(lst))]
}

func NewList[T any](values ...T) *List[T] {
	return &List[T]{
		values: values,
		mu:     &sync.RWMutex{},
	}
}
