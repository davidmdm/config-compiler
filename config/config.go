package config

import (
	"fmt"

	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

func Parse(data []byte) (*Config, error) {
	var c Config
	return &c, yaml.Unmarshal(data, &c)
}

type Config struct {
	Version    string               `yaml:"version"`
	Jobs       map[string]Job       `yaml:"jobs"`
	Workflows  map[string]Workflow  `yaml:"workflows"`
	Setup      bool                 `yaml:"setup,omitempty"`
	Orbs       map[string]string    `yaml:"orbs,omitempty"`
	Commands   map[string]Command   `yaml:"commands,omitempty"`
	Parameters map[string]Parameter `yaml:"parameters,omitempty"`
	Executors  map[string]Executor  `yaml:"executors,omitempty"`
}

func (c Config) ToYAML() ([]byte, error) { return yaml.Marshal(c) }

func (c Config) Compile(pipelineParams map[string]any) (*Config, error) {
	compiled := Config{Version: "2"}

	c.Workflows = make(map[string]Workflow, len(c.Workflows))

	for wfName, wf := range c.Workflows {
		if (wf.When != nil && !wf.When.Evaluate()) || (wf.Unless != nil && wf.Unless.Evaluate()) {
			continue
		}

		for i, wfJob := range wf.Jobs {
			definition, ok := c.Jobs[wfJob.Key]
			if !ok {
				return nil, fmt.Errorf("workflow %q job at index %d references job definition %q that does not exist", wfName, i, wfJob.Key)
			}
			if errs := validateParameters(definition.Parameters, wfJob.Params); len(errs) > 0 {
				return nil, PrettyIndentErr{Message: "error instantiating workflow %q job %q", Errors: errs}
			}
		}

		compiled.Workflows[wfName] = wf
	}

	return nil, nil
}

func validateParameters(parameters map[string]Parameter, values ParamValues) (errs []error) {
	var missingArgs []string
	for name, parameter := range parameters {
		value, ok := values.Lookup(name)
		if parameter.Default == nil && !ok {
			missingArgs = append(missingArgs, name)
			continue
		}
		if actualType := value.GetType(); parameter.Type != actualType {
			if parameter.Type == "enum" && !slices.Contains(parameter.Enum, value.value) {
				errs = append(errs, ParamEnumMismatchErr{
					Name:    name,
					Targets: parameter.Enum,
					Value:   value.value,
				})
			} else {
				errs = append(errs, ParamTypeMismatchErr{
					Name: name,
					Want: parameter.Type,
					Got:  actualType,
				})
			}
		}
	}

	if len(missingArgs) > 0 {
		errs = append(errs, MissingParamsErr(missingArgs))
	}

	return
}
