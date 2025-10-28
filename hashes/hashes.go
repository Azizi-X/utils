package hashes

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"hash"
	"hash/crc32"
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
	if value.Kind() != reflect.Pointer {
		return fmt.Errorf("hashStruct: value must be a pointer to a struct or slice of structs")
	}

	value = value.Elem()

	switch value.Kind() {
	case reflect.Struct:
		return hashSingleStruct(hash, value)
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			item := value.Index(i)
			if item.Kind() == reflect.Ptr {
				item = item.Elem()
			}
			if item.Kind() != reflect.Struct {
				return fmt.Errorf("hashStruct: slice must contain structs or pointers to structs")
			}
			if err := hashSingleStruct(hash, item); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("hashStruct: value must be a pointer to a struct or slice of structs")
	}
}

func hashSingleStruct(hash hash.Hash, value reflect.Value) error {
	fields := make([]fieldData, 0)

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		fieldType := value.Type().Field(i)

		if fieldType.PkgPath != "" {
			continue
		}

		hash_tag := fieldType.Tag.Get("hashes")

		switch hash_tag {
		case "-", "ignore", "false":
			continue
		}

		if hash_tag == "" {
			switch fieldType.Tag.Get("json") {
			case "-":
				continue
			}
		}

		for field.Kind() == reflect.Pointer && !field.IsNil() {
			field = field.Elem()
		}

		switch field.Kind() {
		case reflect.Slice, reflect.Map, reflect.Func:
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

func ShortHash(input string, length int) string {
	hash := fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(input)))
	length = min(length, len(hash))

	return hash[:length]
}
