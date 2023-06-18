package config

import (
	"bytes"
	"regexp"

	"github.com/aymerick/raymond"
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

	handlebarTmpl := toHandlebars(template.String(), expr)

	raw, err := raymond.Render(handlebarTmpl, params)
	if err != nil {
		return nil, err
	}

	// _ = os.WriteFile("./output.template.debug", template.Bytes(), 0o777)
	// _ = os.WriteFile("./output.handlebard.debug", []byte(handlebarTmpl), 0o777)
	// _ = os.WriteFile("./output.debug", []byte(raw), 0o777)

	dst := new(T)
	if err := yaml.Unmarshal([]byte(raw), dst); err != nil {
		return nil, err
	}

	return dst, nil
}
