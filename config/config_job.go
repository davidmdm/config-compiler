package config

import (
	"errors"
	"reflect"

	"gopkg.in/yaml.v3"
)

type Job struct {
	Environment Environment          `yaml:"environment,omitempty"`
	Parallelism int                  `yaml:"parallelism,omitempty"`
	Parameters  map[string]Parameter `yaml:"parameters,omitempty"`
	Steps       []Step               `yaml:"steps"`

	Executor       JobExecutor `yaml:"executor,omitempty"`
	InlineExecutor `yaml:",inline"`

	// name is used for tracking final name in compilation process
	name string
}

type InlineExecutor Executor

type JobExecutor struct {
	Name        string
	ParamValues ParamValues
}

func (executor *JobExecutor) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&executor.Name); err == nil {
		return nil
	}

	executor.ParamValues.parent = reflect.TypeOf(executor)
	if err := node.Decode(&executor.ParamValues); err != nil {
		return nil
	}

	name := executor.ParamValues.Values["name"].String
	if name == "" {
		return errors.New("invalid job executor: name required")
	}

	executor.Name = name

	return nil
}

func (executor JobExecutor) MarshalYAML() (any, error) {
	if len(executor.ParamValues.Values) == 0 {
		return executor.Name, nil
	}
	return executor.ParamValues.Values, nil
}
