package utils

import (
	"encoding/json"
	"reflect"
)

func MergeDefaults(oldJSON, newJSON []byte) ([]byte, error) {
	var old map[string]any
	var new map[string]any

	if err := json.Unmarshal(oldJSON, &old); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(newJSON, &new); err != nil {
		return nil, err
	}

	mergeDefaults(old, new)

	return json.Marshal(old)
}

func mergeDefaults(old, new map[string]any) {
	for key, newValue := range new {
		oldValue, exists := old[key]

		if !exists {
			old[key] = newValue
			continue
		}

		oldMap, oldIsObject := oldValue.(map[string]any)
		newMap, newIsObject := newValue.(map[string]any)

		if oldIsObject && newIsObject {
			mergeDefaults(oldMap, newMap)
			continue
		}

		if reflect.TypeOf(oldValue) != reflect.TypeOf(newValue) {
			old[key] = newValue
		}
	}
}
