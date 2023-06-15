package config

import "gopkg.in/yaml.v3"

type Environment map[string]any

func (env *Environment) UnmarshalYAML(node *yaml.Node) error {
	var envslice []Environment
	if err := node.Decode(&envslice); err == nil {
		for _, envmap := range envslice {
			for k, v := range envmap {
				(*env)[k] = v
			}
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
