package config

import (
	"errors"
	"reflect"

	"gopkg.in/yaml.v3"
)

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
