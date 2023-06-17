package config

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/aymerick/raymond"
	"github.com/davidmdm/yaml"
)

type Parameter struct {
	Description string `yaml:"description,omitempty"`
	Type        string `yaml:"type"`
	Default     any    `yaml:"default,omitempty"`
	Enum        []any  `yaml:"enum,omitempty"`
}

type Parameters map[string]Parameter

func (params Parameters) JoinDefaults(values map[string]any) map[string]any {
	result := map[string]any{}
	for k, v := range params {
		if x, ok := values[k]; ok {
			result[k] = x
		} else {
			result[k] = v.Default
		}
	}
	return result
}

type ParamValues struct {
	Values map[string]ParamValue
	parent reflect.Type
}

func toParamValues(m map[string]any) ParamValues {
	result := ParamValues{Values: make(map[string]ParamValue, len(m))}
	for k, v := range m {
		result.Values[k] = ParamValue{value: v}
	}
	return result
}

func (params ParamValues) AsMap() map[string]any {
	result := make(map[string]any, len(params.Values))
	for k, v := range params.Values {
		if t := v.GetType(); t == "executor" || t == "steps" {
			raw, _ := yaml.Marshal(v.value)
			var x any
			yaml.Unmarshal(raw, &x)
			raw, _ = json.Marshal(x)
			result[k] = raymond.SafeString(raw)
		} else {
			result[k] = v.value
		}
	}
	return result
}

func (params ParamValues) Lookup(name string) (ParamValue, bool) {
	if params.Values == nil {
		return ParamValue{}, false
	}

	result, ok := params.Values[name]
	return result, ok
}

func (params ParamValues) MarshalYAML() (any, error) { return params.Values, nil }

func (param *ParamValues) UnmarshalYAML(node *yaml.Node) error {
	if param.parent == nil {
		return node.Decode(&param.Values)
	}

	var intermediate map[string]any
	if err := node.Decode(&intermediate); err != nil {
		return err
	}

	for _, key := range topLevelKeys(param.parent) {
		delete(intermediate, key)
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

func (param ParamValue) GetType() string {
	switch param.value.(type) {
	case nil:
		return "nil"
	case string:
		return "string"
	case bool:
		return "boolean"
	case int:
		return "integer"
	}
	if len(param.Steps) > 0 {
		return "steps"
	}
	return "executor"
}

func (param *ParamValue) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&param.Integer); err == nil {
		param.value = param.Integer
		return nil
	}
	if err := node.Decode(&param.Boolean); err == nil {
		param.value = param.Boolean
		return nil
	}
	if err := node.Decode(&param.String); err == nil {
		param.value = param.String
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
