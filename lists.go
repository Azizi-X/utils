package utils

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"
)

var (
	IgnoreInvalidListTypes = true
)

type listCore[T any] struct {
	mu     sync.RWMutex
	limit  *int
	equal  func(a, b T) bool
	values []T
}

type List[T any] struct {
	rawCore *listCore[T]
}

func (lst *List[T]) Reverse() {
	if lst == nil {
		return
	}

	core := lst.init()

	core.mu.Lock()
	slices.Reverse(core.values)
	core.mu.Unlock()
}

func (lst *List[T]) RandomItem() *T {
	if lst == nil {
		return nil
	}

	core := lst.init()

	core.mu.RLock()
	randomItem := GetRandomItem(core.values)
	core.mu.RUnlock()
	return randomItem
}

func (core *listCore[T]) checkLimit() {
	if core.limit == nil {
		return
	}

	for len(core.values) > *core.limit {
		core.values = core.values[1:]
	}
}

func (lst *List[T]) SetEqual(fn func(a, b T) bool) *List[T] {
	if lst == nil {
		return nil
	}

	core := lst.init()

	core.mu.Lock()
	core.equal = fn
	core.mu.Unlock()

	return lst
}

func (lst *List[T]) SetLimit(limit int) *List[T] {
	if lst == nil {
		return nil
	}

	core := lst.init()

	core.mu.Lock()
	core.limit = &limit
	core.checkLimit()
	core.mu.Unlock()

	return lst
}

func (lst *List[T]) Contains(value T) bool {
	if lst == nil {
		return false
	}

	core := lst.init()

	core.mu.RLock()

	for i := range core.values {
		if (core.equal != nil && core.equal(core.values[i], value)) ||
			reflect.DeepEqual(core.values[i], value) {
			core.mu.RUnlock()
			return true
		}
	}

	core.mu.RUnlock()
	return false
}

func (lst *List[T]) Last() (T, bool) {
	if lst == nil {
		var empty T
		return empty, false
	}

	core := lst.init()

	core.mu.RLock()

	if len(core.values) > 0 {
		core.mu.RUnlock()
		return core.values[len(core.values)-1], true
	}

	core.mu.RUnlock()

	var empty T
	return empty, false
}

func (lst *List[T]) GetIndex(index int) (T, bool) {
	if lst != nil {
		core := lst.init()
		core.mu.RLock()
		if index >= 0 && index < len(core.values) {
			core.mu.RUnlock()
			return core.values[index], true
		}
		core.mu.RUnlock()
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

	core := lst.init()

	core.mu.RLock()
	length := len(core.values)
	core.mu.RUnlock()
	return length
}

func (lst *List[T]) Sort(fn func(i, j T) bool) {
	if lst == nil {
		return
	}

	core := lst.init()

	core.mu.Lock()
	sort.Slice(core.values, func(i, j int) bool {
		return fn(core.values[i], core.values[j])
	})
	core.mu.Unlock()
}

func (lst *List[T]) GetListClear() (values []T) {
	if lst == nil {
		return []T{}
	}

	core := lst.init()
	core.mu.Lock()
	values = append(values, core.values...)
	core.values = []T{}
	core.mu.Unlock()
	return
}

func (lst *List[T]) Join(sep string) string {
	if lst == nil {
		return ""
	}

	core := lst.init()
	core.mu.RLock()

	var v []string
	for i := range core.values {
		v = append(v, fmt.Sprintf("%v", core.values[i]))
	}

	core.mu.RUnlock()
	return strings.Join(v, sep)
}

func (lst *List[T]) RawList() (values []T) {
	if lst == nil {
		return []T{}
	}

	core := lst.init()
	core.mu.RLock()
	values = core.values
	core.mu.RUnlock()
	return values
}

func (lst *List[T]) GetList() (values []T) {
	if lst == nil {
		return []T{}
	}

	core := lst.init()
	core.mu.RLock()
	values = append(values, core.values...)
	core.mu.RUnlock()
	return
}

func (lst *List[T]) GetFunc(fn func(T) bool) (T, bool) {
	if lst == nil {
		var empty T
		return empty, false
	}

	core := lst.init()
	core.mu.RLock()
	for i := range core.values {
		if fn(core.values[i]) {
			core.mu.RUnlock()
			return core.values[i], true
		}
	}

	core.mu.RUnlock()
	var empty T
	return empty, false
}

func (lst *List[T]) Collect(fn func(T) bool) (values []T) {
	if lst == nil {
		return []T{}
	}

	core := lst.init()
	core.mu.RLock()

	for i := range core.values {
		if fn(core.values[i]) {
			values = append(values, core.values[i])
		}
	}

	core.mu.RUnlock()
	return
}

func (lst *List[T]) AppendFunc(value T, fn func(item T) (exists bool)) (added bool) {
	if lst == nil {
		return false
	}
	core := lst.init()
	core.mu.Lock()

	if slices.ContainsFunc(core.values, fn) {
		core.mu.Unlock()
		return false
	}

	core.values = append(core.values, value)
	core.checkLimit()
	core.mu.Unlock()
	return true
}

func (lst *List[T]) appendDeepEqual(core *listCore[T], value T) bool {
	if core == nil {
		return false
	}

	core.mu.Lock()
	for i := range core.values {
		if reflect.DeepEqual(core.values[i], value) {
			core.mu.Unlock()
			return false
		}
	}

	core.values = append(core.values, value)
	core.checkLimit()
	core.mu.Unlock()

	return true
}

func (lst *List[T]) AppendUnique(value T) bool {
	if lst == nil {
		return false
	}

	core := lst.init()

	if core.equal == nil {
		return lst.appendDeepEqual(core, value)
	}

	core.mu.Lock()

	for i := range core.values {
		if core.equal(core.values[i], value) {
			core.mu.Unlock()
			return false
		}
	}

	core.values = append(core.values, value)
	core.checkLimit()

	core.mu.Unlock()
	return true
}

func (lst *List[T]) ContainsFunc(fn func(value T) bool) bool {
	if lst == nil {
		return false
	}

	core := lst.init()

	core.mu.RLock()
	defer core.mu.RUnlock()

	return slices.ContainsFunc(core.values, fn)
}

func (lst *List[T]) DeleteFuncList(fn func(value T) bool) []T {
	if lst == nil {
		return nil
	}

	core := lst.init()

	core.mu.Lock()
	defer core.mu.Unlock()

	var removed []T

	remove := func(value T) bool {
		remove := fn(value)

		if remove {
			removed = append(removed, value)
		}

		return remove
	}

	core.values = slices.DeleteFunc(core.values, remove)

	return removed
}

func (lst *List[T]) DeleteFunc(fn func(value T) bool) bool {
	if lst == nil {
		return false
	}

	core := lst.init()

	core.mu.Lock()
	defer core.mu.Unlock()

	before := len(core.values)
	core.values = slices.DeleteFunc(core.values, fn)

	return before != len(core.values)
}

func (lst *List[T]) Remove(value T) []T {
	core := lst.init()

	return lst.DeleteFuncList(func(v T) bool {
		if core.equal != nil {
			return core.equal(v, value)
		}
		return reflect.DeepEqual(v, value)
	})
}

func (lst *List[T]) ReplaceFunc(fn func(value T) *T, new ...T) bool {
	if lst == nil {
		return false
	}

	core := lst.init()

	core.mu.Lock()

	var found bool

	for i := range core.values {
		if new := fn(core.values[i]); new != nil {
			found = true
			core.values[i] = *new
		}
	}

	if !found {
		core.values = append(core.values, new...)
	}

	core.checkLimit()
	core.mu.Unlock()

	return found
}

func (lst *List[T]) Modify(fn func(value *T) bool) []T {
	if lst == nil {
		return nil
	}

	core := lst.init()

	core.mu.Lock()

	var modified []T
	for i := range core.values {
		if fn(&core.values[i]) {
			modified = append(modified, core.values[i])
		}
	}

	core.checkLimit()
	core.mu.Unlock()

	return modified
}

func (lst *List[T]) Insert(index int, values ...T) {
	if lst == nil {
		return
	}

	core := lst.init()

	core.mu.Lock()

	index = max(index, 0)
	index = min(index, len(core.values))

	core.values = append(core.values, make([]T, len(values))...)
	copy(core.values[index+len(values):], core.values[index:])
	copy(core.values[index:], values)

	core.checkLimit()
	core.mu.Unlock()
}

func (lst *List[T]) Append(values ...T) {
	if lst == nil {
		return
	}

	core := lst.init()

	core.mu.Lock()
	core.values = append(core.values, values...)
	core.checkLimit()
	core.mu.Unlock()
}

func (lst *List[T]) SetList(values []T) {
	if lst == nil {
		return
	}
	core := lst.init()

	core.mu.Lock()
	core.values = values
	core.checkLimit()
	core.mu.Unlock()
}

func (lst *List[T]) Clear() {
	if lst == nil {
		return
	}
	core := lst.init()

	core.mu.Lock()
	core.values = []T{}
	core.mu.Unlock()
}

func (lst *List[T]) IsZero() bool {
	core := lst.init()

	core.mu.RLock()
	defer core.mu.RUnlock()

	return len(core.values) == 0
}

func (lst List[T]) MarshalJSON() ([]byte, error) {
	core := lst.init()

	core.mu.Lock()
	defer core.mu.Unlock()

	return json.Marshal(core.values)
}

func (lst *List[T]) UnmarshalJSON(data []byte) error {
	if lst == nil {
		return nil
	}

	core := lst.init()

	core.mu.Lock()
	defer core.mu.Unlock()

	var values []T
	if err := json.Unmarshal(data, &values); err != nil {
		if _, ok := err.(*json.UnmarshalTypeError); ok && IgnoreInvalidListTypes {
			fmt.Println("[LIST] Ignoring invalid:", err)
			return nil
		}

		return err
	}

	core.values = values
	return nil
}

func (lst *List[T]) init() *listCore[T] {
	addr := (*unsafe.Pointer)(unsafe.Pointer(&lst.rawCore))
	core := atomic.LoadPointer(addr)

	if core == nil {
		newCore := unsafe.Pointer(newListCore[T]())
		atomic.CompareAndSwapPointer(addr, nil, newCore)
		return lst.init()
	}

	return (*listCore[T])(core)
}

func GetRandomItem[T any](lst []T) *T {
	if len(lst) == 0 {
		return nil
	}

	return &lst[rand.Intn(len(lst))]
}

func newListCore[T any](values ...T) *listCore[T] {
	return &listCore[T]{
		values: values,
	}
}

func NewList[T any](values ...T) *List[T] {
	return &List[T]{
		rawCore: newListCore(values...),
	}
}
