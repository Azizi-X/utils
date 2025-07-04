package hashes

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"hash"
	"reflect"
	"strings"
)

type fieldData struct {
	Name  string
	Value string
}

func UniqueCode(data any, length int) (string, error) {
	hash := sha256.New()

	if err := hashStruct(hash, data); err != nil {
		return "", err
	}

	encoded := base64.RawURLEncoding.EncodeToString(hash.Sum(nil))

	var code string

	if len(encoded) >= length {
		code = encoded[:length]
	} else {
		code = encoded
	}

	return strings.ReplaceAll(code, "_", ""), nil
}

func hashStruct(hash hash.Hash, data any) error {
	value := reflect.ValueOf(data)
	if value.Kind() != reflect.Ptr || value.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("hashStruct: value must be a pointer to a struct")
	}

	value = value.Elem()

	fields := make([]fieldData, 0)

	for i := range value.NumField() {
		field := value.Field(i)
		fieldType := value.Type().Field(i)

		switch field.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Func:
			continue
		}

		fields = append(fields, fieldData{
			Name:  fieldType.Name,
			Value: fmt.Sprintf("%v", field.Interface()),
		})
	}

	for _, field := range fields {
		if _, err := hash.Write([]byte(field.Name)); err != nil {
			return err
		}
		if _, err := hash.Write([]byte(field.Value)); err != nil {
			return err
		}
	}

	return nil
}
