package config

import (
	"fmt"
	"reflect"

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

	// var m map[string]RawNode
	// if err := node.Decode(&m); err != nil {

	// }

	var shorthandRun struct {
		Run string `yaml:"run"`
	}

	if err := node.Decode(&shorthandRun); err == nil && shorthandRun.Run != "" {
		step.Type = "run"
		step.Run.Command = shorthandRun.Run
		return nil
	}

	var runStmt struct {
		Run RunData `yaml:"run"`
	}
	if err := node.Decode(&runStmt); err == nil && !reflect.ValueOf(runStmt).IsZero() {
		step.Type = "run"
		step.Run = runStmt.Run
		return nil
	}

	var whenStmt struct {
		When Conditional `yaml:"when"`
	}
	if err := node.Decode(&whenStmt); err == nil && !reflect.ValueOf(whenStmt).IsZero() {
		step.Type = "when"
		step.When = whenStmt.When
		return nil
	}

	var unlessStmt struct {
		Unless Conditional `yaml:"unless"`
	}
	if err := node.Decode(&unlessStmt); err == nil && !reflect.ValueOf(unlessStmt).IsZero() {
		step.Type = "unless"
		step.When = unlessStmt.Unless
		return nil
	}

	var elem map[string]ParamValues
	if err := node.Decode(&elem); err != nil {
		return err
	}
	if len(elem) != 1 {
		return fmt.Errorf("expected step have single key but got: %d", len(elem))
	}

	for key, data := range elem {
		step.Type = key
		step.params = data
	}

	return nil
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
