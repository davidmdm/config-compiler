package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type Command struct {
	Description string               `yaml:"description,omitempty"`
	Parameters  map[string]Parameter `yaml:"parameters,omitempty"`
	Steps       []Step               `yaml:"steps"`
}

type RunData struct {
	Command         string      `yaml:"command"`
	Name            string      `yaml:"name,omitempty"`
	Shell           string      `yaml:"shell,omitempty"`
	Environment     Environment `yaml:"environment,omitempty"`
	Background      bool        `yaml:"background,omitempty"`
	WorkDir         string      `yaml:"working_directory,omitempty"`
	NoOutputTimeout string      `yaml:"no_output_timeout,omitempty"`
	When            string      `yaml:"when,omitempty"`
}

type Step struct {
	Type   string      `yaml:"-"`
	Run    RunData     `yaml:"run,omitempty"`
	When   Conditional `yaml:"when,omitempty"`
	Unless Conditional `yaml:"unless,omitempty"`
	params ParamValues `yaml:"-"`
}

func (step *Step) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&step.Type); err == nil {
		return nil
	}

	var m map[string]RawNode
	if err := node.Decode(&m); err != nil {
		return err
	}

	if len(m) != 1 {
		return fmt.Errorf("step can only be of one type")
	}

	for key, value := range m {
		step.Type = key
		node = value.Node
	}

	switch step.Type {
	case "run":
		if err := node.Decode(&step.Run.Command); err == nil {
			return nil
		}
		return node.Decode(&step.Run)
	case "when":
		return node.Decode(&step.When)
	case "unless":
		return node.Decode(&step.Unless)
	default:
		return node.Decode(&step.params)
	}
}

func (step Step) MarshalYAML() (any, error) {
	switch step.Type {
	case "run":
		return map[string]RunData{"run": step.Run}, nil
	case "when":
		return map[string]Conditional{"when": step.When}, nil
	case "unless":
		return map[string]Conditional{"unless": step.Unless}, nil
	}

	if len(step.params.Values) == 0 {
		return step.Type, nil
	}

	return map[string]ParamValues{step.Type: step.params}, nil
}
