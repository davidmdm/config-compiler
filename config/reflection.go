package config

import (
	"reflect"
	"strings"

	"golang.org/x/exp/slices"
)

type yamlTags struct {
	Name      string
	Inline    bool
	OmitEmpty bool
}

func getYAMLTags(f reflect.StructField) yamlTags {
	tags := strings.Split(f.Tag.Get("yaml"), ",")
	return yamlTags{
		Name:      tags[0],
		Inline:    slices.Contains(tags[1:], "inline"),
		OmitEmpty: slices.Contains(tags[1:], "omitempty"),
	}
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
		tags := getYAMLTags(field)
		if tags.Name == "-" {
			continue
		}
		if tags.Inline {
			result = append(result, topLevelKeys(field.Type)...)
			continue
		}
		result = append(result, tags.Name)
	}

	return result
}

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
		tags := getYAMLTags(f)
		if tags.Name == "-" {
			continue
		}
		if tags.Inline {
			structToMap(v.Field(i), m)
			continue
		}
		if tags.OmitEmpty && v.Field(i).IsZero() {
			continue
		}
		m[tags.Name] = v.Field(i).Interface()
	}

	return m
}

func asAnyMap[T any](m map[string]T) map[string]any {
	target := make(map[string]any, len(m))
	for k, v := range m {
		target[k] = v
	}
	return target
}
