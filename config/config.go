package config

import (
	"bytes"
	"fmt"
	"reflect"
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

type JobCacheItem struct {
	Instantiations []Job
	Parameters     map[string]Parameter
}

func Compile(source []byte, pipelineParams map[string]any) (*Config, error) {
	var rootNode RawNode
	if err := yaml.Unmarshal(source, &rootNode); err != nil {
		return nil, err
	}

	parameters, err := getParametersFromNode(rootNode.Node)
	if err != nil {
		return nil, err
	}

	if errs := validateParameters(parameters, toParamValues(pipelineParams)); len(errs) > 0 {
		return nil, PrettyIndentErr{Message: "pipeline parameter error(s):", Errors: errs}
	}

	if node, err := applyPipelineParams[RawNode](rootNode.Node, pipelineParams); err != nil {
		return nil, err
	} else {
		rootNode = *node
	}

	var root struct {
		Setup     bool                `yaml:"setup"`
		Orbs      map[string]string   `yaml:"orbs"`
		Workflows map[string]Workflow `yaml:"workflows"`
		Jobs      map[string]RawNode  `yaml:"jobs"`
		Commands  map[string]RawNode  `yaml:"commands"`
		Executors map[string]RawNode  `yaml:"executors"`
	}

	if err := rootNode.Decode(&root); err != nil {
		return nil, err
	}

	state := struct {
		Jobs          map[string][]*Job
		Workflows     map[string][]*Job
		WorkflowNames []string
	}{
		Jobs:          map[string][]*Job{},
		Workflows:     map[string][]*Job{},
		WorkflowNames: []string{},
	}

	// First pass through workflows simply validates the workflows reference valid jobs, and that the
	// parameters are aligned. It only evaluates workflows that will not be skipped. If a workflow is valid
	// it is written to the compiled version for future processing.
	for workflowName, workflow := range root.Workflows {
		if (workflow.When != nil && !workflow.When.Evaluate()) || (workflow.Unless != nil && workflow.Unless.Evaluate()) {
			continue
		}

		state.WorkflowNames = append(state.WorkflowNames, workflowName)

		for i, workflowJob := range workflow.Jobs {
			definition, ok := root.Jobs[workflowJob.Key]
			if !ok {
				return nil, fmt.Errorf("workflow %q job at index %d references job definition %q that does not exist", workflowName, i, workflowJob.Key)
			}

			parameters, err := getParametersFromNode(definition.Node)
			if err != nil {
				return nil, err
			}

			if errs := validateParameters(parameters, workflowJob.Params); len(errs) > 0 {
				return nil, PrettyIndentErr{Message: "parameter error(s) instantiating workflow at %q.%q:", Errors: errs}
			}

			job, err := applyParams[Job](definition.Node, workflowJob.Params.AsMap())
			if err != nil {
				return nil, err
			}

			if job.Executor.Name != "" {
				exNode, ok := root.Executors[job.Executor.Name]
				if !ok {
					return nil, fmt.Errorf("executor not found: %s", job.Executor.Name)
				}
				ex, err := applyParams[Executor](exNode.Node, job.Executor.ParamValues.AsMap())
				if err != nil {
					return nil, err
				}
				job.InlineExecutor = InlineExecutor(*ex)
			}

			var steps []Step
			steps = append(steps, workflowJob.PreSteps...)
			steps = append(steps, job.Steps...)
			steps = append(steps, workflowJob.PostSteps...)

			job.Steps, err = expandMultiStep(root.Commands, steps)
			if err != nil {
				return nil, err
			}

			jobName := workflowJob.Name
			if jobName == "" {
				jobName = workflowJob.Key
			}

			jobIdx := slices.IndexFunc(state.Jobs[jobName], func(j *Job) bool {
				return reflect.DeepEqual(job, j)
			})

			if jobIdx < 0 {
				state.Jobs[jobName] = append(state.Jobs[jobName], job)
			} else {
				job = state.Jobs[jobName][jobIdx]
			}

			state.Workflows[workflowName] = append(state.Workflows[workflowName], job)
		}
	}

	compiled := Config{
		Version:   "2",
		Jobs:      map[string]Job{},
		Workflows: map[string]Workflow{},
	}

	for name, jobs := range state.Jobs {
		jobTotal := len(jobs)
		for i, job := range jobs {
			if jobTotal == 1 {
				job.name = name
			} else {
				job.name = fmt.Sprintf("%s-%d", name, i+1)
			}
			// zero out reusable fields
			job.Parameters = nil
			job.Executor = JobExecutor{}
			compiled.Jobs[job.name] = *job
		}
	}

	for name, jobs := range state.Workflows {
		workflowJobs := make([]WorkflowJob, len(jobs))
		for i, j := range jobs {
			workflowJobs[i] = WorkflowJob{Key: j.name}
		}

		targetWorkflow := root.Workflows[name]
		targetWorkflow.Unless = nil
		targetWorkflow.When = nil
		targetWorkflow.Jobs = workflowJobs

		compiled.Workflows[name] = targetWorkflow
	}

	return &compiled, nil
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

var (
	paramExpr         = regexp.MustCompile(`<<(\s*parameters\.\w+)\s*>>`)
	pipelineParamExpr = regexp.MustCompile(`<<\s*pipeline\.parameters\.\w+\s*>>`)
)

func toHandlebars(source string, expr *regexp.Regexp) string {
	return expr.ReplaceAllStringFunc(source, func(s string) string {
		raw := []byte(s)
		raw[0], raw[1], raw[len(raw)-2], raw[len(raw)-1] = '{', '{', '}', '}'
		return string(raw)
	})
}

func applyParams[T any](node *yaml.Node, params map[string]any) (*T, error) {
	return apply[T](node, paramExpr, map[string]any{"parameters": params})
}

func applyPipelineParams[T any](node *yaml.Node, params map[string]any) (*T, error) {
	return apply[T](node, pipelineParamExpr, map[string]any{"pipeline": map[string]any{"parameters": params}})
}

func apply[T any](node *yaml.Node, expr *regexp.Regexp, params map[string]any) (*T, error) {
	var template bytes.Buffer
	if err := yaml.NewEncoder(&template).Encode(node); err != nil {
		return nil, err
	}

	handlebarTmpl := toHandlebars(template.String(), expr)

	raw, err := raymond.Render(handlebarTmpl, params)
	if err != nil {
		return nil, err
	}

	dst := new(T)
	return dst, yaml.Unmarshal([]byte(raw), dst)
}

func toParamValues(m map[string]any) ParamValues {
	result := ParamValues{Values: make(map[string]ParamValue, len(m))}
	for k, v := range m {
		result.Values[k] = ParamValue{value: v}
	}
	return result
}

func getParametersFromNode(node *yaml.Node) (map[string]Parameter, error) {
	var parameterNode struct {
		Parameters map[string]Parameter `yaml:"parameters"`
	}
	if err := node.Decode(&parameterNode); err != nil {
		return nil, err
	}
	return parameterNode.Parameters, nil
}

func expandMultiStep(commands map[string]RawNode, steps []Step) ([]Step, error) {
	var result []Step
	for _, substep := range steps {
		if substeps, err := expandStep(commands, substep); err != nil {
			return nil, err
		} else {
			result = append(result, substeps...)
		}
	}
	return result, nil
}

func expandStep(commands map[string]RawNode, step Step) ([]Step, error) {
	switch {
	case step.Type == "when":
		if !step.When.Condition.Evaluate() {
			return nil, nil
		}
		return expandMultiStep(commands, step.When.Steps)
	case step.Type == "unless":
		if !step.Unless.Condition.Evaluate() {
			return nil, nil
		}
		return expandMultiStep(commands, step.Unless.Steps)
	case slices.Contains(stepCmds, step.Type):
		return []Step{step}, nil
	default:
		cmdNode, ok := commands[step.Type]
		if !ok {
			return nil, fmt.Errorf("command not found: %s", step.Type)
		}

		parameters, err := getParametersFromNode(cmdNode.Node)
		if err != nil {
			return nil, err
		}

		if errs := validateParameters(parameters, step.Params); len(errs) > 0 {
			return nil, PrettyIndentErr{Message: fmt.Sprintf("parameter error(s) invoking command %s", step.Type), Errors: errs}
		}

		cmd, err := applyParams[Command](cmdNode.Node, step.Params.AsMap())
		if err != nil {
			return nil, err
		}

		return expandMultiStep(commands, cmd.Steps)
	}
}
