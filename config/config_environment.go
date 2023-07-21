package config

import (
	"fmt"
	"strings"

	"github.com/davidmdm/yaml"
)

type Environment map[string]any

func (env *Environment) UnmarshalYAML(node *yaml.Node) error {
	var envSlice []Environment
	if err := node.Decode(&envSlice); err == nil {
		*env = map[string]any{}
		for _, envmap := range envSlice {
			for k, v := range envmap {
				(*env)[k] = v
			}
		}
		return nil
	}

	var stringSlice []string
	if err := node.Decode(&stringSlice); err == nil {
		*env = map[string]any{}
		for _, s := range stringSlice {

			kvs := strings.SplitN(s, "=", 2)
			if len(kvs) != 2 {
				return fmt.Errorf("environment string should be of form KEY=value, not %s", s)
			}
			(*env)[kvs[0]] = kvs[1]
		}
		return nil
	}

	var target map[string]any
	if err := node.Decode(&target); err != nil {
		return err
	}

	*env = target
	return nil
}

func (env Environment) MarshalYAML() (any, error) {
	for k, v := range env {
		if v == nil {
			env[k] = ""
		}
	}
	return map[string]any(env), nil
}
