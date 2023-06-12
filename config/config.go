package config

import (
	"fmt"
	"regexp"

	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"

	"github.com/aymerick/raymond"
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

	// jobInstantiations := map[string][]Job{}

	c.Workflows = make(map[string]Workflow, len(c.Workflows))

	// First pass through workflows simply validates the workflows reference valid jobs, and that the
	// parameters are aligned. It only evaluates workflows that will not be skipped. If a workflow is valid
	// it is written to the compiled version for future processing.
	for workflowName, workflow := range c.Workflows {
		if (workflow.When != nil && !workflow.When.Evaluate()) || (workflow.Unless != nil && workflow.Unless.Evaluate()) {
			continue
		}

		for i, workflowJob := range workflow.Jobs {
			definition, ok := c.Jobs[workflowJob.Key]
			if !ok {
				return nil, fmt.Errorf("workflow %q job at index %d references job definition %q that does not exist", workflowName, i, workflowJob.Key)
			}
			if errs := validateParameters(definition.Parameters, workflowJob.Params); len(errs) > 0 {
				return nil, PrettyIndentErr{Message: "error instantiating workflow at %q.%q:", Errors: errs}
			}

		}

		compiled.Workflows[workflowName] = workflow
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

func instantiateJob(c Config, def Job, params ParamValues) Job {
	return Job{}
}

func (c Config) instantiateCommand(cmd Command, params ParamValues) {
}

var paramExpr = regexp.MustCompile(`<<(\s*(pipeline\.)?parameters\.\w+)\s*>>`)

func toHandlebars(source string) string {
	return paramExpr.ReplaceAllStringFunc(source, func(s string) string {
		raw := []byte(s)
		raw[0], raw[1], raw[len(raw)-2], raw[len(raw)-1] = '{', '{', '}', '}'
		return string(raw)
	})
}

func applyParams[T any](value *T, params map[string]any) error {
	template, err := yaml.Marshal(value)
	if err != nil {
		return err
	}

	handlebarTmpl := toHandlebars(string(template))

	raw, err := raymond.Render(handlebarTmpl, map[string]any{"parameters": params})
	if err != nil {
		return err
	}

	return yaml.Unmarshal([]byte(raw), value)
}
