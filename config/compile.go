package config

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/davidmdm/yaml"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
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

type ApprovalJob struct {
	Offset int
	Job    WorkflowJob
}

type MatrixJob struct {
	MatrixValues []KV
	Job          *Job
}

type WFJob struct {
	Requires StringList
	Contexts StringList
	Filters  Filters
	*Job
}

type compilerState struct {
	Jobs      map[string][]MatrixJob
	Workflows map[string][]WFJob
	Approvals map[string][]ApprovalJob
}

type Compiler struct {
	root struct {
		Setup     bool               `yaml:"setup"`
		Workflows Workflows          `yaml:"workflows"`
		Orbs      map[string]string  `yaml:"orbs"`
		Jobs      map[string]RawNode `yaml:"jobs"`
		Commands  map[string]RawNode `yaml:"commands"`
		Executors map[string]RawNode `yaml:"executors"`
	}

	orbs Orbs

	state compilerState

	// GetOrbSource defines how orb data will be fetched.
	GetOrbSource func(ref string) (string, error)
}

func (c Compiler) Compile(source []byte, pipelineParams map[string]any) ([]byte, error) {
	if c.GetOrbSource == nil {
		c.GetOrbSource = GetOrbSource
	}

	c.state = compilerState{
		Jobs:      map[string][]MatrixJob{},
		Workflows: map[string][]WFJob{},
		Approvals: map[string][]ApprovalJob{},
	}

	var rootNode RawNode
	if err := yaml.Unmarshal(source, &rootNode); err != nil {
		return nil, fmt.Errorf("invalid source: %v", err)
	}

	resolveAliases(rootNode.Node)

	parameters, err := getParametersFromNode(rootNode.Node)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline parameter definition: %v", err)
	}

	var pipelineParameters map[string]any
	if pipelineParams != nil {
		value, ok := pipelineParams["parameters"]
		if ok {
			params, ok := value.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("failed to parse provided pipeline parameters: `parameters` key must have a map[string]any value")
			}
			pipelineParameters = params
		}
	}

	if errs := validateParameters(parameters, toParamValues(pipelineParameters)); len(errs) > 0 {
		return nil, PrettyErr{Message: "pipeline parameter error(s):", Errors: errs}
	}

	if pipelineParams == nil {
		pipelineParams = map[string]any{"parameters": parameters.JoinDefaults(map[string]any{})}
	} else {
		pipelineParams["parameters"] = parameters.JoinDefaults(pipelineParameters)
	}

	if len(pipelineParams) > 0 {
		if node, err := applyPipelineParams[RawNode](rootNode.Node, pipelineParams); err != nil {
			return nil, err
		} else {
			rootNode = *node
		}
	}

	if err := rootNode.Decode(&c.root); err != nil {
		return nil, err
	}

	c.orbs = make(Orbs, len(c.root.Orbs))

	for name, orb := range c.root.Orbs {
		src, err := c.GetOrbSource(orb)
		if err != nil {
			return nil, fmt.Errorf("failed to get orb: %s", orb)
		}

		src = strings.ReplaceAll(src, "{{", "<<")
		src = strings.ReplaceAll(src, "}}", ">>")

		var raw Orb
		if err := yaml.Unmarshal([]byte(src), &raw); err != nil {
			return nil, fmt.Errorf("failed to parse orb %s: %w", name, err)
		}
		c.orbs[name] = raw
	}

	// First pass through workflows simply validates the workflows reference valid jobs, and that the
	// parameters are aligned. It only evaluates workflows that will not be skipped. If a workflow is valid
	// it is written to the compiled version for future processing.
	for name, workflow := range c.root.Workflows {
		if err := c.processWorkflow(name, workflow); err != nil {
			return nil, fmt.Errorf("error processing workflow %s: %v", name, err)
		}
	}

	cfg := c.compile()

	return yaml.Marshal(cfg)
}

func (c Compiler) compile() Config {
	compiled := Config{
		Version:   2,
		Jobs:      map[string]Job{},
		Workflows: map[string]Workflow{},
	}

	nameMapping := map[string][]string{}

	for name, matrixJobs := range c.state.Jobs {
		jobTotal := len(matrixJobs)
		compiledNames := make([]string, jobTotal)
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

			compiledNames[i] = job.name

			// zero out reusable fields
			job.Parameters = nil
			job.Executor = JobExecutor{}
			compiled.Jobs[job.name] = *job
		}
		nameMapping[name] = compiledNames
	}

	for name, jobs := range c.state.Workflows {
		workflowJobs := make([]WorkflowJob, len(jobs))
		for i, j := range jobs {

			var requires []string
			for _, name := range j.Requires {
				requires = append(requires, nameMapping[name]...)
			}

			workflowJobs[i] = WorkflowJob{
				Key: j.name,
				WorkflowJobData: WorkflowJobData{
					WorkflowJobProps: WorkflowJobProps{
						Requires: requires,
						Context:  j.Contexts,
						Filters:  j.Filters,
					},
				},
			}
		}

		targetWorkflow := c.root.Workflows[name]
		targetWorkflow.When = nil
		targetWorkflow.Jobs = workflowJobs

		for i, approval := range c.state.Approvals[name] {
			var requires []string
			for _, name := range approval.Job.Requires {
				requires = append(requires, nameMapping[name]...)
			}
			approval.Job.Requires = requires
			targetWorkflow.Jobs = slices.Insert(targetWorkflow.Jobs, approval.Offset+i, approval.Job)
		}

		compiled.Workflows[name] = targetWorkflow
	}

	return compiled
}

func (c Compiler) processWorkflow(name string, workflow Workflow) error {
	if !workflow.When.Evaluate() {
		return nil
	}

	var (
		offset           int
		workflowJobNames = make([]string, 0, len(workflow.Jobs))
	)

	for _, workflowJob := range workflow.Jobs {
		workflowJobNames = append(workflowJobNames, workflowJob.Name())

		if workflowJob.Type == "approval" {
			c.state.Approvals[name] = append(c.state.Approvals[name], ApprovalJob{
				Offset: offset,
				Job:    workflowJob,
			})
			continue
		}

		jobNode, ok := c.root.Jobs[workflowJob.Key]
		if !ok {
			jobNode, ok = c.orbs.GetJobNode(workflowJob.Key)
			if !ok {
				return fmt.Errorf("job %s not found", workflowJob.Key)
			}
		}

		matrixKVs := flattenKeyedMatrix(workflowJob.Matrix.Parameters)

		if len(matrixKVs) == 0 {
			matrixKVs = make([][]KV, 1)
		}

		for _, matrix := range matrixKVs {
			offset += 1
			if err := c.processJob(name, workflowJob, matrix, jobNode.Node); err != nil {
				return fmt.Errorf("job %s: %v", workflowJob.Key, err)
			}
		}
	}

	var errs []error

	for _, wfJob := range workflow.Jobs {
		for _, required := range wfJob.Requires {
			if !slices.Contains(workflowJobNames, required) {
				errs = append(errs, fmt.Errorf("job %s cannot require %s: no job named %s in workflow", wfJob.Name(), required, required))
			}
		}
	}

	if len(errs) > 0 {
		return PrettyErr{Message: "job requirement error(s):", Errors: errs}
	}

	return nil
}

func (c *Compiler) processJob(workflowName string, workflowJob WorkflowJob, matrix []KV, jobNode *yaml.Node) error {
	parameters, err := getParametersFromNode(jobNode)
	if err != nil {
		return err
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
		return PrettyErr{Message: "parameter error(s):", Errors: errs}
	}

	job, err := applyParams[Job](jobNode, parameters.JoinDefaults(paramValues.AsMap()))
	if err != nil {
		return err
	}

	if job.Executor.Name != "" {
		exNode, ok := c.root.Executors[job.Executor.Name]
		if !ok {
			exNode, ok = c.orbs.GetExecutorNode(job.Executor.Name)
			if !ok {
				return fmt.Errorf("executor not found: %s", job.Executor.Name)
			}
		}

		parameters, err := getParametersFromNode(exNode.Node)
		if err != nil {
			return err
		}

		ex, err := applyParams[Executor](exNode.Node, parameters.JoinDefaults(job.Executor.ParamValues.AsMap()))
		if err != nil {
			return err
		}
		job.InlineExecutor = InlineExecutor(*ex)
	}

	var steps []Step
	steps = append(steps, workflowJob.PreSteps...)
	steps = append(steps, job.Steps...)
	steps = append(steps, workflowJob.PostSteps...)

	job.Steps, err = c.expandMultiStep("", steps)
	if err != nil {
		return err
	}

	if len(job.Steps) == 0 {
		return errors.New("steps are required but got none")
	}

	jobName := workflowJob.Name()

	jobIdx := slices.IndexFunc(c.state.Jobs[jobName], func(j MatrixJob) bool {
		return reflect.DeepEqual(job, j.Job)
	})

	if jobIdx < 0 {
		c.state.Jobs[jobName] = append(c.state.Jobs[jobName], MatrixJob{
			MatrixValues: matrix,
			Job:          job,
		})
	} else {
		job = c.state.Jobs[jobName][jobIdx].Job
	}

	c.state.Workflows[workflowName] = append(c.state.Workflows[workflowName], WFJob{
		Requires: workflowJob.Requires,
		Contexts: workflowJob.Context,
		Filters:  workflowJob.Filters,
		Job:      job,
	})

	return nil
}

func (c Compiler) expandMultiStep(orbCtx string, steps []Step) ([]Step, error) {
	var (
		result []Step
		errs   []error
	)
	for i, substep := range steps {
		if substeps, err := c.expandStep(orbCtx, substep); err != nil {
			stepName := substep.Type
			if orbCtx != "" {
				stepName = orbCtx + "/" + stepName
			}
			errs = append(errs, fmt.Errorf("step %d: %s: %w", i, stepName, err))
		} else {
			result = append(result, substeps...)
		}
	}
	if len(errs) > 0 {
		return nil, OrderedErr{Message: "could not compile step(s):", Errors: errs}
	}
	return result, nil
}

func (c Compiler) expandStep(orbCtx string, step Step) ([]Step, error) {
	switch {
	case step.Type == "when":
		if step.When == nil || !step.When.Condition.Evaluate() {
			return nil, nil
		}
		return c.expandMultiStep(orbCtx, step.When.Steps)
	case step.Type == "unless":
		if step.Unless == nil || step.Unless.Condition.Evaluate() {
			return nil, nil
		}
		return c.expandMultiStep(orbCtx, step.Unless.Steps)
	case slices.Contains(stepCmds, step.Type):
		return []Step{step}, nil
	default:
		cmdNode, ok := c.root.Commands[step.Type]
		if !ok {
			cmdNode, orbCtx, ok = c.orbs.GetCommandNode(orbCtx, step.Type)
			if !ok {
				return nil, fmt.Errorf("command not found")
			}
		}

		parameters, err := getParametersFromNode(cmdNode.Node)
		if err != nil {
			return nil, err
		}

		if errs := validateParameters(parameters, step.Params); len(errs) > 0 {
			return nil, PrettyErr{Message: fmt.Sprintf("parameter error(s) invoking command %s", step.Type), Errors: errs}
		}

		cmd, err := applyParams[Command](cmdNode.Node, parameters.JoinDefaults(step.Params.AsMap()))
		if err != nil {
			return nil, err
		}

		return c.expandMultiStep(orbCtx, cmd.Steps)
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

func getParametersFromNode(node *yaml.Node) (Parameters, error) {
	var parameterNode struct {
		Parameters Parameters `yaml:"parameters"`
	}
	if err := node.Decode(&parameterNode); err != nil {
		return nil, err
	}
	return parameterNode.Parameters, nil
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

	for name := range values.Values {
		if _, ok := parameters[name]; !ok {
			errs = append(errs, fmt.Errorf("unknown argument: %s", name))
		}
	}

	if len(missingArgs) > 0 {
		errs = append(errs, MissingParamsErr(missingArgs))
	}

	return
}
