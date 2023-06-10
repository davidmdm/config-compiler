package config

import "gopkg.in/yaml.v3"

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
