package config

import (
	"fmt"
	"reflect"

	"github.com/davidmdm/yaml"
	"golang.org/x/exp/slices"
)

type Command struct {
	Description string               `yaml:"description,omitempty"`
	Parameters  map[string]Parameter `yaml:"parameters,omitempty"`
	Steps       []Step               `yaml:"steps"`
}

type Run struct {
	Command         string      `yaml:"command"`
	Name            string      `yaml:"name,omitempty"`
	Shell           StringList  `yaml:"shell,omitempty"`
	Environment     Environment `yaml:"environment,omitempty"`
	Background      bool        `yaml:"background,omitempty"`
	WorkDir         string      `yaml:"working_directory,omitempty"`
	NoOutputTimeout string      `yaml:"no_output_timeout,omitempty"`
	When            RunWhen     `yaml:"when,omitempty"`
}

type RunWhen string

func (when *RunWhen) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode((*string)(when)); err != nil {
		return err
	}

	switch string(*when) {
	case "always", "on_success", "on_fail":
		return nil
	default:
		return fmt.Errorf("invalid when attribute: wanted one of always, on_success, or on_fail but got: %s", *when)
	}
}

type Checkout struct {
	Path string `yaml:"path,omitempty"`
}

type SetupRemoteDocker struct {
	// TODO - check recent changes
	DockerLayerCaching bool   `yaml:"docker_layer_caching,omitempty"`
	Version            string `yaml:"version,omitempty"`
}

type SaveCache struct {
	Paths []string `yaml:"paths"`
	Key   string   `yaml:"key"`
	Name  string   `yaml:"name,omitempty"`
	When  string   `yaml:"when,omitempty"`
}

type RestoreCache struct {
	Key  string   `yaml:"key,omitempty"`
	Keys []string `yaml:"keys,omitempty"`
	Name string   `yaml:"name,omitempty"`
}

type StoreArtifacts struct {
	Path        string `yaml:"path"`
	Destination string `yaml:"destination,omitempty"`
	Name        string `yaml:"name,omitempty"`
}

type StoreTestResults struct {
	Path string `yaml:"path"`
}

type PersistToWorkspace struct {
	Root  string   `yaml:"root"`
	Paths []string `yaml:"paths"`
	Name  string   `yaml:"name,omitempty"`
}

type AttachWorkspace struct {
	At   string `yaml:"at"`
	Name string `yaml:"name,omitempty"`
}

type AddSSHKeys struct {
	Fingerprints []string `yaml:"fingerprints"`
}

type StepCMD struct {
	Run                Run                `yaml:"run,omitempty"`
	Checkout           Checkout           `yaml:"checkout,omitempty"`
	SetupRemoteDocker  SetupRemoteDocker  `yaml:"setup_remote_docker,omitempty"`
	SaveCache          SaveCache          `yaml:"save_cache,omitempty"`
	RestoreCache       RestoreCache       `yaml:"restore_cache,omitempty"`
	StoreArtifacts     StoreArtifacts     `yaml:"store_artifacts,omitempty"`
	StoreTestResults   StoreTestResults   `yaml:"store_test_results,omitempty"`
	PersistToWorkspace PersistToWorkspace `yaml:"persist_to_workspace,omitempty"`
	AttachWorkspace    AttachWorkspace    `yaml:"attach_workspace,omitempty"`
	AddSSHKeys         AddSSHKeys         `yaml:"add_ssh_keys,omitempty"`
	When               *ConditionalSteps  `yaml:"when,omitempty"`
	Unless             *ConditionalSteps  `yaml:"unless,omitempty"`
}

type Step struct {
	Type    string      `yaml:"-"`
	Params  ParamValues `yaml:"-"`
	StepCMD `yaml:",inline"`
}

var stepCmds = topLevelKeys(reflect.TypeOf(StepCMD{}))

func (step *Step) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&step.Type); err == nil {
		return nil
	}

	var m map[string]RawNode
	if err := node.Decode(&m); err != nil {
		return err
	}

	if len(m) != 1 {
		return fmt.Errorf("step can only be of one type")
	}

	var childNode *yaml.Node
	for key, value := range m {
		step.Type = key
		childNode = value.Node
	}

	if slices.Contains(stepCmds, step.Type) {
		// Attempt short run declaration
		if step.Type == "run" && childNode.Decode(&step.Run.Command) == nil {
			return nil
		}
		return node.Decode(&step.StepCMD)
	}

	return childNode.Decode(&step.Params)
}

func (step Step) MarshalYAML() (any, error) {
	if slices.Contains(stepCmds, step.Type) {
		if step.Type == "checkout" && (step.Checkout == Checkout{}) {
			return "checkout", nil
		}
		if step.Type == "setup_remote_docker" && (step.SetupRemoteDocker == SetupRemoteDocker{}) {
			return "setup_remote_docker", nil
		}
		return step.StepCMD, nil
	}

	if len(step.Params.Values) == 0 {
		return step.Type, nil
	}

	return map[string]ParamValues{step.Type: step.Params}, nil
}
