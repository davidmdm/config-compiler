package config

import (
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"
)

type Workflow struct {
	Jobs []WorkflowJob `yaml:"jobs"`
	When *Condition    `yaml:"when,omitempty"`
}

func (workflow *Workflow) UnmarshalYAML(node *yaml.Node) error {
	var state struct {
		Jobs   []WorkflowJob `yaml:"jobs"`
		Unless *Condition    `yaml:"unless"`
		When   *Condition    `yaml:"when"`
	}
	if err := node.Decode(&state); err != nil {
		return err
	}

	if len(state.Jobs) == 0 {
		return fmt.Errorf("workflow must contain at least one job")
	}

	workflow.Jobs = state.Jobs

	if state.Unless != nil && state.When != nil {
		return fmt.Errorf("cannot declare both when and unless at the same time")
	}

	if state.Unless != nil {
		workflow.When = &Condition{SubCondition: SubCondition{Not: state.Unless}}
		return nil
	}

	workflow.When = state.When

	return nil
}

type JobMatrix struct {
	Parameters map[string][]any `yaml:"parameters"`
	Exclude    []map[string]any `yaml:"exclude,omitempty"`
}

type WorkflowJobProps struct {
	Name      string     `yaml:"name,omitempty"`
	Type      string     `yaml:"type,omitempty"`
	Requires  StringList `yaml:"requires,omitempty"`
	Context   StringList `yaml:"context,omitempty"`
	Filters   Filters    `yaml:"filters,omitempty"`
	Matrix    JobMatrix  `yaml:"matrix,omitempty"`
	PreSteps  []Step     `yaml:"pre-steps,omitempty"`
	PostSteps []Step     `yaml:"post-steps,omitempty"`
}

type WorkflowJobData struct {
	WorkflowJobProps `yaml:",inline"`
	Params           ParamValues `yaml:"-"`
}

func (wfjd *WorkflowJobData) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&wfjd.WorkflowJobProps); err != nil {
		return err
	}
	wfjd.Params.parent = reflect.TypeOf(wfjd)
	return node.Decode(&wfjd.Params)
}

func (wfjd WorkflowJobData) MarshalYAML() (any, error) {
	return structToMap(wfjd.WorkflowJobProps, asAnyMap(wfjd.Params.Values)), nil
}

type WorkflowJob struct {
	Key             string `yaml:"-"`
	WorkflowJobData `yaml:",inline"`
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

func (job WorkflowJob) MarshalYAML() (any, error) {
	if reflect.ValueOf(job.WorkflowJobData).IsZero() {
		return job.Key, nil
	}
	return map[string]WorkflowJobData{job.Key: job.WorkflowJobData}, nil
}

type Filters struct {
	Branches FilterConditions `yaml:"branches,omitempty"`
	Tags     FilterConditions `yaml:"tags,omitempty"`
}

type FilterConditions struct {
	Only   StringList `yaml:"only,omitempty"`
	Ignore StringList `yaml:"ignore,omitempty"`
}
