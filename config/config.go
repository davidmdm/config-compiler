package config

import (
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
