package config

import (
	"fmt"
	"strings"
)

type ParamTypeMismatchErr struct {
	Name string
	Want string
	Got  string
}

func (err ParamTypeMismatchErr) Error() string {
	return fmt.Sprintf("type mismatch for param %s: wanted %s but got %s", err.Name, err.Want, err.Got)
}

type ParamEnumMismatchErr struct {
	Name    string
	Targets []any
	Value   any
}

func (err ParamEnumMismatchErr) Error() string {
	targets := make([]string, len(err.Targets))
	for i, elem := range err.Targets {
		targets[i] = fmt.Sprint(elem)
	}

	return fmt.Sprintf(
		"enum mismatch for param %s: wanted one of (%s) but got %v",
		err.Name, strings.Join(targets, ", "), err.Value,
	)
}

type MissingParamsErr []string

func (err MissingParamsErr) Error() string {
	return fmt.Sprintf("missing required parameters: %s", strings.Join(err, ", "))
}

type PrettyIndentErr struct {
	Message string
	Errors  []error
}

func (err PrettyIndentErr) Error() string {
	indentedErrors := make([]string, len(err.Errors))
	for i, e := range err.Errors {
		indentedErrors[i] = indent(e.Error())
	}
	return err.Message + "\n" + strings.Join(indentedErrors, "\n")
}

func indent(value string) string {
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = "\t" + line
	}
	return strings.Join(lines, "\n")
}
