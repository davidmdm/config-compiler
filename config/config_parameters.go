package config

import (
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

type Parameter struct {
	Description string `yaml:"description,omitempty"`
	Type        string `yaml:"type"`
	Default     any    `yaml:"default"`
	Enum        []any  `yaml:"enum,omitempty"`
}

type ParamValues struct {
	Values map[string]ParamValue
	parent reflect.Type
}

func (param *ParamValues) UnmarshalYAML(node *yaml.Node) error {
	if param.parent == nil {
		return node.Decode(&param.Values)
	}

	typ := param.parent
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	var intermediate map[string]any
	if err := node.Decode(&intermediate); err != nil {
		return err
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		name := strings.Split(field.Tag.Get("yaml"), ",")[0]
		delete(intermediate, name)
	}
	data, err := yaml.Marshal(intermediate)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &param.Values)
}

type ParamValue struct {
	String   string
	Integer  int
	Boolean  bool
	Steps    []Step
	Executor JobExecutor

	value any
}

func (param *ParamValue) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&param.String); err == nil {
		param.value = param.String
		return nil
	}
	if err := node.Decode(&param.Integer); err == nil {
		param.value = param.Integer
		return nil
	}
	if err := node.Decode(&param.Boolean); err == nil {
		param.value = param.Boolean
		return nil
	}
	if err := node.Decode(&param.Steps); err == nil {
		param.value = param.Steps
		return nil
	}
	if err := node.Decode(&param.Executor); err == nil {
		param.value = param.Executor
		return nil
	}

	var v any
	_ = node.Decode(&v)

	return fmt.Errorf("invalid param value: %v", v)
}

func (param ParamValue) MarshalYAML() (any, error) {
	return param.value, nil
}
