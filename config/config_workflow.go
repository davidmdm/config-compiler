package config

import (
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"
)

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

// WorkflowJobData is embedded in WorkflowJob and acts as a layer to avoid recursiveness in yaml decoding.
// It allows us to separate the data from the key when jobs are in the format: {"key": {"props"...}}
type WorkflowJobData struct {
	Name      string     `yaml:"name,omitempty"`
	Type      string     `yaml:"type,omitempty"`
	Requires  StringList `yaml:"requires,omitempty"`
	Context   StringList `yaml:"context,omitempty"`
	Filters   Filters    `yaml:"filters,omitempty"`
	Matrix    JobMatrix  `yaml:"matrix,omitempty"`
	PreSteps  []Step     `yaml:"pre-steps,omitempty"`
	PostSteps []Step     `yaml:"post-steps,omitempty"`
}

type WorkflowJob struct {
	Key             string `yaml:"-"`
	WorkflowJobData `yaml:",inline"`
	Values          ParamValues `yaml:"-"`
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

	job.Values.parent = reflect.TypeOf(job)

	return node.Decode(&job.Values)
}

type Filters struct {
	Branches FilterConditions `yaml:"branches,omitempty"`
	Tags     FilterConditions `yaml:"tags,omitempty"`
}

type FilterConditions struct {
	Only   StringList `yaml:"only,omitempty"`
	Ignore StringList `yaml:"ignore,omitempty"`
}
