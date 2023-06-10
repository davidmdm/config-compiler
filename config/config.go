package config

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

func Parse(data []byte) (*Config, error) {
	var c Config
	return &c, yaml.Unmarshal(data, &c)
}

type Config struct {
	Version    string               `yaml:"version"`
	Setup      bool                 `yaml:"setup,omitempty"`
	Jobs       map[string]Job       `yaml:"jobs"`
	Workflows  map[string]Workflow  `yaml:"workflows"`
	Orbs       map[string]string    `yaml:"orbs,omitempty"`
	Commands   map[string]Command   `yaml:"commands,omitempty"`
	Parameters map[string]Parameter `yaml:"parameters,omitempty"`
	Executors  map[string]Executor  `yaml:"executors,omitempty"`
}

func (c Config) ToYAML() ([]byte, error) { return yaml.Marshal(c) }

func (c Config) Compile(pipelineParams map[string]any) ([]byte, error) { return yaml.Marshal(c) }

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

type Command struct {
	Description string               `yaml:"description,omitempty"`
	Parameters  map[string]Parameter `yaml:"parameters,omitempty"`
	Steps       []Step               `yaml:"steps"`
}

type Parameter struct {
	Description string `yaml:"description,omitempty"`
	Type        string `yaml:"type"`
	Default     any    `yaml:"default"`
	Enum        []any  `yaml:"enum,omitempty"`
}

type ParamValues struct {
	Values map[string]ParamValue
	parent reflect.Type
}

func (param *ParamValues) UnmarshalYAML(node *yaml.Node) error {
	if param.parent == nil {
		return node.Decode(&param.Values)
	}

	typ := param.parent
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	var intermediate map[string]any
	if err := node.Decode(&intermediate); err != nil {
		return err
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		name := strings.Split(field.Tag.Get("yaml"), ",")[0]
		delete(intermediate, name)
	}
	data, err := yaml.Marshal(intermediate)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &param.Values)
}

type ParamValue struct {
	String   string
	Integer  int
	Boolean  bool
	Steps    []Step
	Executor JobExecutor

	value any
}

func (param *ParamValue) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&param.String); err == nil {
		param.value = param.String
		return nil
	}
	if err := node.Decode(&param.Integer); err == nil {
		param.value = param.Integer
		return nil
	}
	if err := node.Decode(&param.Boolean); err == nil {
		param.value = param.Boolean
		return nil
	}
	if err := node.Decode(&param.Steps); err == nil {
		param.value = param.Steps
		return nil
	}
	if err := node.Decode(&param.Executor); err == nil {
		param.value = param.Executor
		return nil
	}

	var v any
	_ = node.Decode(&v)

	return fmt.Errorf("invalid param value: %v", v)
}

func (param ParamValue) MarshalYAML() (any, error) {
	return param.value, nil
}

type Executor struct {
	ResourceClass string   `yaml:"resource_class,omitempty"`
	Docker        []Docker `yaml:"docker,omitempty"`
	MacOS         MacOS    `yaml:"macos,omitempty"`
	Machine       Machine  `yaml:"machine,omitempty"`
}

type Machine struct {
	Image              string `yaml:"image"`
	DockerLayerCaching bool   `yaml:"docker_layer_caching,omitempty"`
}

type Docker struct {
	Image       string      `yaml:"image"`
	Name        string      `yaml:"name,omitempty"`
	EntryPoint  StringList  `yaml:"entrypoint,omitempty"`
	Command     StringList  `yaml:"command,omitempty"`
	User        string      `yaml:"user,omitempty"`
	Environment Environment `yaml:"environment,omitempty"`
	Auth        Auth        `yaml:"auth,omitempty"`
}

type Auth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type MacOS struct {
	XCode string `yaml:"xcode"`
}

type Job struct {
	Environment Environment          `yaml:"environment,omitempty"`
	Parallelism int                  `yaml:"parallelism,omitempty"`
	Parameters  map[string]Parameter `yaml:"parameters,omitempty"`
	Steps       []Step               `yaml:"steps"`

	Executor       JobExecutor `yaml:"executor,omitempty"`
	InlineExecutor `yaml:",inline"`
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

type Matches struct {
	Pattern string `yaml:"pattern"`
	Value   string `yaml:"value"`
}
type SubCondition struct {
	And     []Condition `yaml:"and,omitempty"`
	Or      []Condition `yaml:"or,omitempty"`
	Equal   []Condition `yaml:"equal,omitempty"`
	Not     *Condition  `yaml:"not,omitempty"`
	Matches Matches     `yaml:"matches,omitempty"`
}

type Condition struct {
	Literal      any `yaml:"-"`
	SubCondition `yaml:",inline"`
}

func (cond Condition) MarshalYAML() (any, error) {
	if !reflect.ValueOf(cond.SubCondition).IsZero() {
		return cond.SubCondition, nil
	}
	return cond.Literal, nil
}

func (cond *Condition) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&cond.SubCondition); err == nil {
		return nil
	}

	initializedFields := 0
	item := reflect.ValueOf(cond.SubCondition)

	for i := 0; i < item.NumField(); i++ {
		if item.Field(i).IsZero() {
			continue
		}
		initializedFields++
	}

	if initializedFields > 1 {
		return errors.New("only one of [and, or, equal, not, matches] can be defined")
	}

	return node.Decode(&cond.Literal)
}

type Conditional struct {
	Condition Condition `yaml:"condition"`
	Steps     []Step    `yaml:"steps"`
}

type Step struct {
	Run    RunData     `yaml:"run,omitempty"`
	When   Conditional `yaml:"when,omitempty"`
	Unless Conditional `yaml:"unless,omitempty"`
	Type   string      `yaml:"-"`
	Values ParamValues `yaml:"-"`
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

	if len(step.Values.Values) == 0 {
		return step.Type, nil
	}

	return map[string]ParamValues{step.Type: step.Values}, nil
}

func (step *Step) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&step.Type); err == nil {
		return nil
	}

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
		step.Values = data
	}

	return nil
}

type Workflow struct {
	Jobs   WorkflowJobs `yaml:"jobs"`
	When   Conditional  `yaml:"when,omitempty"`
	Unless Conditional  `yaml:"unless,omitempty"`
}

type WorkflowJobs []WorkflowJob

func (jobs WorkflowJobs) MarshalYAML() (any, error) {
	target := make([]map[string]WorkflowJobData, len(jobs))
	for i, job := range jobs {
		target[i] = map[string]WorkflowJobData{job.Key: job.WorkflowJobData}
	}
	return target, nil
}

type JobMatrix struct {
	Parameters map[string][]any `yaml:"parameters"`
	Exclude    []map[string]any `yaml:"exclude,omitempty"`
}

type WorkflowJob struct {
	Key string `yaml:"-"`
	WorkflowJobData
}

// WorkflowJobData is embedded in WorkflowJob and acts as a layer to avoid recursiveness in yaml decoding.
// It allows us to separate the data from the key when jobs are in the format: {"key": {"props"...}}
type WorkflowJobData struct {
	Name      string      `yaml:"name,omitempty"`
	Type      string      `yaml:"type,omitempty"`
	Requires  StringList  `yaml:"requires,omitempty"`
	Context   StringList  `yaml:"context,omitempty"`
	Filters   Filters     `yaml:"filters,omitempty"`
	Matrix    JobMatrix   `yaml:"matrix,omitempty"`
	PreSteps  []Step      `yaml:"pre-steps,omitempty"`
	PostSteps []Step      `yaml:"post-steps,omitempty"`
	Values    ParamValues `yaml:"-"`
}

func (job *WorkflowJob) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&job.Key); err == nil {
		return nil
	}

	var elem map[string]WorkflowJobData
	if err := node.Decode(&elem); err != nil {
		return err
	}
	if len(elem) != 1 {
		return fmt.Errorf("expected single key in workflow job definition but got: %d", len(elem))
	}

	for key, data := range elem {
		job.Key = key
		job.WorkflowJobData = data
	}

	return nil
}

type Filters struct {
	Branches FilterConditions `yaml:"branches,omitempty"`
	Tags     FilterConditions `yaml:"tags,omitempty"`
}

type FilterConditions struct {
	Only   StringList `yaml:"only,omitempty"`
	Ignore StringList `yaml:"ignore,omitempty"`
}

type StringList []string

func (list *StringList) UnmarshalYAML(node *yaml.Node) error {
	return decodeOneOrMore(node, list)
}

func (list StringList) MarshalYAML() (any, error) {
	if len(list) == 1 {
		return list[0], nil
	}
	return []string(list), nil
}

func decodeOneOrMore[T any, V ~[]T, P *V](node *yaml.Node, pointer P) error {
	var single T
	if err := node.Decode(&single); err == nil {
		*pointer = []T{single}
		return nil
	}
	var many []T
	if err := node.Decode(&many); err != nil {
		return err
	}
	*pointer = many
	return nil
}
