package config

import (
	"bytes"
	"errors"
	"regexp"
	"strings"

	"github.com/davidmdm/handlebars"
	"github.com/davidmdm/yaml"
)

var (
	paramExpr         = regexp.MustCompile(`<<(\s*parameters\.[\w-]+)\s*>>`)
	pipelineParamExpr = regexp.MustCompile(`<<\s*pipeline\.[\w-]+(\.[\w-]+)*\s*>>`)
)

func toHandlebars(source string, expr *regexp.Regexp) string {
	return expr.ReplaceAllStringFunc(source, func(s string) string {
		raw := []byte(s)
		raw[0], raw[1], raw[len(raw)-2], raw[len(raw)-1] = '{', '{', '}', '}'
		return string(raw)
	})
}

func applyParams[T any](node *yaml.Node, params map[string]any) (*T, error) {
	return apply[T](node, paramExpr, map[string]any{"parameters": params})
}

func applyPipelineParams[T any](node *yaml.Node, params map[string]any) (*T, error) {
	return apply[T](node, pipelineParamExpr, map[string]any{"pipeline": params})
}

func apply[T any](node *yaml.Node, expr *regexp.Regexp, params map[string]any) (*T, error) {
	var template bytes.Buffer
	if err := yaml.NewEncoder(&template).Encode(node); err != nil {
		return nil, err
	}

	raw, err := func() ([]byte, error) {
		if expr == nil {
			return template.Bytes(), nil
		}
		tpl, err := handlebars.Parse(toHandlebars(template.String(), expr))
		if err != nil {
			return nil, err
		}

		var errs []error

	outer:
		for _, expr := range tpl.ExpressionPaths() {
			if !strings.HasPrefix(expr, "pipeline.") && !strings.HasPrefix(expr, "parameters.") {
				continue
			}
			current := params
			for _, pathSegment := range strings.Split(expr, ".") {
				value, ok := current[pathSegment]
				if !ok {
					errs = append(errs, errors.New(expr))
					continue outer
				}
				current, _ = value.(map[string]any)
			}
		}

		if len(errs) > 0 {
			return nil, PrettyErr{Message: "argument(s) referenced in template but not declared:", Errors: errs}
		}

		raw, err := tpl.Exec(params)
		return []byte(raw), err
	}()
	if err != nil {
		return nil, err
	}

	// _ = os.WriteFile("./output.template.debug", template.Bytes(), 0o777)
	// _ = os.WriteFile("./output.handlebard.debug", []byte(handlebarTmpl), 0o777)
	// _ = os.WriteFile("./output.debug", []byte(raw), 0o777)

	var dst T
	if err := yaml.Unmarshal(raw, &dst); err != nil {
		return nil, err
	}

	return &dst, nil
}
