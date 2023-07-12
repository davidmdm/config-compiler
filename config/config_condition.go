package config

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"

	"github.com/davidmdm/yaml"
)

type subexpressionErr string

func (err subexpressionErr) Error() string {
	return string(err)
}

func (subexpressionErr) Is(err error) bool {
	_, ok := err.(subexpressionErr)
	return ok
}

type ConditionalSteps struct {
	Condition Condition `yaml:"condition"`
	Steps     []Step    `yaml:"steps"`
}

type Matches struct {
	Pattern *Expression `yaml:"pattern"`
	Value   string      `yaml:"value"`
}

type Expression regexp.Regexp

func (expr *Expression) UnmarshalYAML(node *yaml.Node) error {
	var raw string
	if err := node.Decode(&raw); err != nil {
		return err
	}
	if len(raw) == 0 {
		return errors.New("pattern cannot be empty")
	}
	if raw[0] == '/' && raw[len(raw)-1] == '/' {
		raw = raw[1:raw[len(raw)-1]]
	}

	expression, err := regexp.Compile(raw)
	if err != nil {
		return subexpressionErr(fmt.Sprintf("failed to compile pattern: %v", err))
	}

	*expr = Expression(*expression)
	return nil
}

func (expr Expression) MarshalYAML() (any, error) {
	return fmt.Sprintf("/%v/", expr), nil
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
	} else if errors.Is(err, subexpressionErr("")) {
		return err
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

func (cond *Condition) Evaluate() bool {
	if cond == nil {
		return true
	}

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

	if expr := (*regexp.Regexp)(cond.Matches.Pattern); expr != nil && expr.String() != "" {
		return expr.MatchString(cond.Matches.Value)
	}

	return !reflect.ValueOf(cond.Literal).IsZero()
}
