package config

import (
	"fmt"
	"reflect"
	"strings"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Version    float64              `yaml:"version"`
	Jobs       map[string]Job       `yaml:"jobs"`
	Workflows  map[string]Workflow  `yaml:"workflows"`
	Setup      bool                 `yaml:"setup,omitempty"`
	Orbs       map[string]string    `yaml:"orbs,omitempty"`
	Commands   map[string]Command   `yaml:"commands,omitempty"`
	Parameters map[string]Parameter `yaml:"parameters,omitempty"`
	Executors  map[string]Executor  `yaml:"executors,omitempty"`
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

	orbs := make(Orbs, len(root.Orbs))

	for name, orb := range root.Orbs {
		src, err := GetOrbSource(orb)
		if err != nil {
			return nil, fmt.Errorf("failed to get orb: %s", orb)
		}

		src = strings.ReplaceAll(src, "{{", "<<")
		src = strings.ReplaceAll(src, "}}", ">>")

		var raw Orb
		if err := yaml.Unmarshal([]byte(src), &raw); err != nil {
			return nil, fmt.Errorf("failed to parse orb %s: %w", name, err)
		}
		orbs[name] = raw
	}

	type ApprovalJob struct {
		Index int
		Job   WorkflowJob
	}

	type MatrixJob struct {
		MatrixValues []KV
		Job          *Job
	}

	state := struct {
		Jobs      map[string][]MatrixJob
		Workflows map[string][]*Job
		Approvals map[string][]ApprovalJob
	}{
		Jobs:      map[string][]MatrixJob{},
		Workflows: map[string][]*Job{},
		Approvals: map[string][]ApprovalJob{},
	}

	// First pass through workflows simply validates the workflows reference valid jobs, and that the
	// parameters are aligned. It only evaluates workflows that will not be skipped. If a workflow is valid
	// it is written to the compiled version for future processing.
	for workflowName, workflow := range root.Workflows {
		if (workflow.When != nil && !workflow.When.Evaluate()) || (workflow.Unless != nil && workflow.Unless.Evaluate()) {
			continue
		}

		for i, workflowJob := range workflow.Jobs {
			if workflowJob.Type == "approval" {
				state.Approvals[workflowName] = append(state.Approvals[workflowName], ApprovalJob{
					Index: i,
					Job:   workflowJob,
				})
				continue
			}

			jobNode, ok := root.Jobs[workflowJob.Key]
			if !ok {
				jobNode, ok = orbs.GetJobNode(workflowJob.Key)
				if !ok {
					return nil, fmt.Errorf("workflow %q job at index %d references job definition %q that does not exist", workflowName, i, workflowJob.Key)
				}
			}

			matrixKVs := flattenMatrix(workflowJob.Matrix.Parameters)

			if len(matrixKVs) == 0 {
				matrixKVs = make([][]KV, 1)
			}

			for _, matrix := range matrixKVs {
				parameters, err := getParametersFromNode(jobNode.Node)
				if err != nil {
					return nil, err
				}

				paramValues := func() ParamValues {
					if len(matrix) == 0 {
						return workflowJob.Params
					}
					values := map[string]ParamValue{}
					for _, kv := range matrix {
						values[kv.Key] = ParamValue{value: kv.Value}
					}
					maps.Copy(values, workflowJob.Params.Values)
					return ParamValues{Values: values}
				}()

				if errs := validateParameters(parameters, paramValues); len(errs) > 0 {
					return nil, PrettyIndentErr{Message: fmt.Sprintf("parameter error(s) instantiating workflow at %s.%s:", workflowName, workflowJob.Key), Errors: errs}
				}

				job, err := applyParams[Job](jobNode.Node, parameters.JoinDefaults(paramValues.AsMap()))
				if err != nil {
					return nil, err
				}

				if job.Executor.Name != "" {
					exNode, ok := root.Executors[job.Executor.Name]
					if !ok {
						exNode, ok = orbs.GetExecutorNode(job.Executor.Name)
						if !ok {
							return nil, fmt.Errorf("executor not found: %s", job.Executor.Name)
						}
					}

					parameters, err := getParametersFromNode(exNode.Node)
					if err != nil {
						return nil, err
					}

					ex, err := applyParams[Executor](exNode.Node, parameters.JoinDefaults(job.Executor.ParamValues.AsMap()))
					if err != nil {
						return nil, err
					}
					job.InlineExecutor = InlineExecutor(*ex)
				}

				var steps []Step
				steps = append(steps, workflowJob.PreSteps...)
				steps = append(steps, job.Steps...)
				steps = append(steps, workflowJob.PostSteps...)

				job.Steps, err = expandMultiStep(root.Commands, orbs, "", steps)
				if err != nil {
					return nil, err
				}

				jobName := workflowJob.Name
				if jobName == "" {
					jobName = workflowJob.Key
				}

				jobIdx := slices.IndexFunc(state.Jobs[jobName], func(j MatrixJob) bool {
					return reflect.DeepEqual(job, j.Job)
				})

				if jobIdx < 0 {
					state.Jobs[jobName] = append(state.Jobs[jobName], MatrixJob{
						MatrixValues: matrix,
						Job:          job,
					})
				} else {
					job = state.Jobs[jobName][jobIdx].Job
				}

				state.Workflows[workflowName] = append(state.Workflows[workflowName], job)
			}
		}
	}

	compiled := Config{
		Version:   2,
		Jobs:      map[string]Job{},
		Workflows: map[string]Workflow{},
	}

	for name, matrixJobs := range state.Jobs {
		jobTotal := len(matrixJobs)
		for i, matrixJob := range matrixJobs {

			job := matrixJob.Job

			if jobTotal == 1 {
				job.name = name
			} else {
				job.name = fmt.Sprintf("%s-%d", name, i+1)
			}

			matrixSuffix := func() string {
				if matrixJob.MatrixValues == nil {
					return ""
				}
				values := []string{}
				for _, kv := range matrixJob.MatrixValues {
					values = append(values, fmt.Sprintf("%v", kv.Value))
				}
				return strings.Join(values, "-")
			}()

			if matrixSuffix != "" {
				job.name = name + "-" + matrixSuffix
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

		for i, approval := range state.Approvals[name] {
			targetWorkflow.Jobs = slices.Insert(targetWorkflow.Jobs, approval.Index+i, approval.Job)
		}

		compiled.Workflows[name] = targetWorkflow
	}

	return &compiled, nil
}

func validateParameters(parameters map[string]Parameter, values ParamValues) (errs []error) {
	var missingArgs []string
	for name, parameter := range parameters {
		value, ok := values.Lookup(name)
		if !ok || value.value == nil {
			if parameter.Default == nil {
				missingArgs = append(missingArgs, name)
			}
			continue
		}

		if actualType := value.GetType(); parameter.Type != actualType {
			switch {
			case parameter.Type == "enum" && !slices.Contains(parameter.Enum, value.value):
				errs = append(errs, ParamEnumMismatchErr{
					Name:    name,
					Targets: parameter.Enum,
					Value:   value.value,
				})

			case parameter.Type == "env_var_name" && actualType == "string":
				continue

			default:
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

func getParametersFromNode(node *yaml.Node) (Parameters, error) {
	var parameterNode struct {
		Parameters Parameters `yaml:"parameters"`
	}
	if err := node.Decode(&parameterNode); err != nil {
		return nil, err
	}
	return parameterNode.Parameters, nil
}

func expandMultiStep(commands map[string]RawNode, orbs Orbs, orbCtx string, steps []Step) ([]Step, error) {
	var result []Step
	for _, substep := range steps {
		if substeps, err := expandStep(commands, orbs, orbCtx, substep); err != nil {
			return nil, err
		} else {
			result = append(result, substeps...)
		}
	}
	return result, nil
}

func expandStep(commands map[string]RawNode, orbs Orbs, orbCtx string, step Step) ([]Step, error) {
	switch {
	case step.Type == "when":
		if !step.When.Condition.Evaluate() {
			return nil, nil
		}
		return expandMultiStep(commands, orbs, orbCtx, step.When.Steps)
	case step.Type == "unless":
		if !step.Unless.Condition.Evaluate() {
			return nil, nil
		}
		return expandMultiStep(commands, orbs, orbCtx, step.Unless.Steps)
	case slices.Contains(stepCmds, step.Type):
		return []Step{step}, nil
	default:
		cmdNode, ok := commands[step.Type]
		if !ok {
			cmdNode, orbCtx, ok = orbs.GetCommandNode(orbCtx, step.Type)
			if !ok {
				return nil, fmt.Errorf("command not found: %s", step.Type)
			}
		}

		parameters, err := getParametersFromNode(cmdNode.Node)
		if err != nil {
			return nil, err
		}

		if errs := validateParameters(parameters, step.Params); len(errs) > 0 {
			return nil, PrettyIndentErr{Message: fmt.Sprintf("parameter error(s) invoking command %s", step.Type), Errors: errs}
		}

		cmd, err := applyParams[Command](cmdNode.Node, parameters.JoinDefaults(step.Params.AsMap()))
		if err != nil {
			return nil, err
		}

		return expandMultiStep(commands, orbs, orbCtx, cmd.Steps)
	}
}

type Orb struct {
	Jobs      map[string]RawNode `yaml:"jobs"`
	Commands  map[string]RawNode `yaml:"commands"`
	Executors map[string]RawNode `yaml:"executors"`
}
type Orbs map[string]Orb

func (orbs Orbs) GetExecutorNode(ref string) (RawNode, bool) {
	before, after, ok := strings.Cut(ref, "/")
	if !ok {
		return RawNode{}, false
	}
	orb, ok := orbs[before]
	if !ok {
		return RawNode{}, false
	}
	node, ok := orb.Executors[after]
	return node, ok
}

func (orbs Orbs) GetJobNode(ref string) (RawNode, bool) {
	before, after, ok := strings.Cut(ref, "/")
	if !ok {
		return RawNode{}, false
	}
	orb, ok := orbs[before]
	if !ok {
		return RawNode{}, false
	}
	node, ok := orb.Jobs[after]
	return node, ok
}

func (orbs Orbs) GetCommandNode(orbCtx, ref string) (RawNode, string, bool) {
	before, after, ok := strings.Cut(ref, "/")
	if !ok {
		before = orbCtx
		after = ref
	}
	orb, ok := orbs[before]
	if !ok {
		return RawNode{}, "", false
	}
	node, ok := orb.Commands[after]
	return node, before, ok
}

type KV struct {
	Key   string
	Value any
}

func flattenMatrix(m map[string][]any) [][]KV {
	keys := maps.Keys(m)
	slices.Sort(keys)

	matrix := make([][]any, len(keys))
	for i, k := range keys {
		matrix[i] = m[k]
	}

	rows := crossProduct(matrix)

	result := make([][]KV, len(rows))
	for i, row := range rows {
		keyValues := make([]KV, len(row))
		for j, v := range row {
			keyValues[j] = KV{
				Key:   keys[j],
				Value: v,
			}
		}
		result[i] = keyValues
	}

	return result
}

func crossProduct[T any](m [][]T) [][]T {
	if len(m) == 0 {
		return nil
	}

	// len(m) parameters, so each row shall be of size len(m).
	rowSize := len(m)

	total := 1
	for _, set := range m {
		total *= len(set)
	}

	subTotals := make([]int, len(m))
	for i := range m {
		subTotals[i] = func() int {
			total := 1
			for _, set := range m[i+1:] {
				total *= len(set)
			}
			return total
		}()
	}

	result := make([][]T, total)

	for i := 0; i < total; i++ {
		row := make([]T, rowSize)
		for pos := 0; pos < rowSize; pos++ {
			row[pos] = m[pos][(i/subTotals[pos])%len(m[pos])]
		}
		result[i] = row
	}

	return result
}
