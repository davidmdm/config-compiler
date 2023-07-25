package config

import (
	"fmt"

	"github.com/davidmdm/yaml"
)

type StringList []string

func (list *StringList) UnmarshalYAML(node *yaml.Node) error {
	return decodeOneOrMore(node, list)
}

func (list StringList) MarshalYAML() (any, error) {
	if len(list) == 1 {
		return list[0], nil
	}
	return []string(list), nil
}

func decodeOneOrMore[T any, V ~[]T, P *V](node *yaml.Node, pointer P) error {
	var single T
	if err := node.Decode(&single); err == nil {
		*pointer = []T{single}
		return nil
	}
	var many []T
	if err := node.Decode(&many); err != nil {
		return err
	}
	*pointer = many
	return nil
}

type RawNode struct{ *yaml.Node }

func (n *RawNode) UnmarshalYAML(node *yaml.Node) error {
	n.Node = node
	return nil
}

func resolveAliases(node *yaml.Node) {
	node.Anchor = ""
	for _, n := range node.Content {
		resolveAliases(n)
	}
	if node.Kind != yaml.AliasNode || node.Alias == nil {
		return
	}
	*node = *node.Alias
}

type List[T any] []T

func (l *List[T]) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.SequenceNode {
		return fmt.Errorf("expected a string but got: %s", node.Tag)
	}
	var (
		results = make([]T, len(node.Content))
		errs    []error
	)

	for i, n := range node.Content {
		if err := n.Decode(&results[i]); err != nil {
			errs = append(errs, fmt.Errorf("position %d: %w", i, err))
		}
	}

	switch len(errs) {
	case 0:
		*l = results
		return nil
	case 1:
		return errs[0]
	default:
		return OrderedPrettyIndentErr{
			Message: "",
			Errors:  errs,
		}
	}
}
