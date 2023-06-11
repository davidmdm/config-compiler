package config

import (
	"reflect"
	"strings"

	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

type StringList []string

func (list *StringList) UnmarshalYAML(node *yaml.Node) error {
	return decodeOneOrMore(node, list)
}

func (list StringList) MarshalYAML() (any, error) {
	if len(list) == 1 {
		return list[0], nil
	}
	return []string(list), nil
}

func decodeOneOrMore[T any, V ~[]T, P *V](node *yaml.Node, pointer P) error {
	var single T
	if err := node.Decode(&single); err == nil {
		*pointer = []T{single}
		return nil
	}
	var many []T
	if err := node.Decode(&many); err != nil {
		return err
	}
	*pointer = many
	return nil
}

func topLevelKeys(typ reflect.Type) []string {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil
	}

	var result []string
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tags := strings.Split(field.Tag.Get("yaml"), ",")

		name := tags[0]
		if name == "-" {
			continue
		}
		if len(tags) > 1 && slices.Contains(tags[1:], "inline") {
			result = append(result, topLevelKeys(field.Type)...)
			continue
		}
		result = append(result, name)
	}

	return result
}

// func toMap(item any) map[string]any {
// 	v := reflect.ValueOf(item)

// 	for v.Kind() == reflect.Pointer {
// 		if v.IsNil() {
// 			return nil
// 		}
// 		v = v.Elem()
// 	}

// 	switch v.Kind() {
// 	case reflect.Map:
// 		result := make(map[string]any, v.Len())
// 		iter := v.MapRange()
// 		for iter.Next() {
// 			result[iter.Key().String()] = iter.Value().Interface()
// 		}
// 		return result
// 	case reflect.Struct:
// 		return structToMap(v, make(map[string]any, v.NumField()))

// 	default:
// 		return nil
// 	}
// }

func structToMap(item any, m map[string]any) map[string]any {
	v := reflect.ValueOf(item)

	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return m
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return m
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tags := strings.Split(f.Tag.Get("yaml"), ",")
		if len(tags) > 1 && slices.Contains(tags[1:], "inline") {
			structToMap(v.Field(i), m)
			continue
		}
		if len(tags) > 1 && slices.Contains(tags[1:], "omitempty") && v.Field(i).IsZero() {
			continue
		}
		m[tags[0]] = v.Field(i).Interface()
	}

	return m
}

func toAnyMap[T any](m map[string]T) map[string]any {
	target := make(map[string]any, len(m))
	for k, v := range m {
		target[k] = v
	}
	return target
}
