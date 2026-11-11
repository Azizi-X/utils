package utils

import (
	"maps"
	"slices"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

type OrderedMap[K comparable, V any] = orderedmap.OrderedMap[K, V]

func NewOrderedMap[V any](values map[string]V, ordering []string) *OrderedMap[string, V] {
	result := orderedmap.New[string, V](len(values))

	for _, key := range ordering {
		if value, ok := values[key]; ok {
			result.Set(key, value)
		}
	}

	if result.Len() == len(values) {
		return result
	}

	for _, key := range slices.Sorted(maps.Keys(values)) {
		if _, ok := result.Get(key); !ok {
			result.Set(key, values[key])
		}
	}

	return result
}

func NewOrderedMapPairs(values []any) *OrderedMap[string, any] {
	if len(values)%2 != 0 {
		panic("ordered map values must be key-value pairs")
	}

	result := orderedmap.New[string, any](len(values) / 2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			panic("ordered map keys must be strings")
		}
		result.Set(key, values[i+1])
	}

	return result
}
