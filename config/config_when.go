package config

import (
	"errors"
	"reflect"
	"regexp"

	"gopkg.in/yaml.v3"
)

type ConditionalSteps struct {
	Condition Condition `yaml:"condition"`
	Steps     []Step    `yaml:"steps"`
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

func (cond Condition) Evaluate() bool {
	if len(cond.And) > 0 {
		for _, subcond := range cond.And {
			if !subcond.Evaluate() {
				return false
			}
		}
		return true
	}

	if len(cond.Or) > 0 {
		for _, subcond := range cond.Or {
			if subcond.Evaluate() {
				return true
			}
		}
		return false
	}

	if cond.Not != nil {
		return !cond.Not.Evaluate()
	}

	if size := len(cond.Equal); size > 0 {
		if size == 1 {
			return true
		}
		for i := 1; i < size; i++ {
			if !reflect.DeepEqual(cond.Equal[i-1], cond.Equal[i]) {
				return false
			}
		}
		return true
	}

	if cond.Matches.Pattern != "" {
		expression, err := regexp.Compile(cond.Matches.Pattern)
		if err != nil {
			panic("invalid pattern: " + cond.Matches.Pattern)
		}
		return expression.MatchString(cond.Matches.Value)
	}

	return !reflect.ValueOf(cond.Literal).IsZero()
}
