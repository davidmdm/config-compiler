package config

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/davidmdm/yaml"
	"github.com/stretchr/testify/require"
)

func TestConfig_Environment(t *testing.T) {
	//t.Skip()

	for _, tt := range []struct {
		name, in string
		expected Environment
	}{
		{
			name: "env vars obj",
			in:   `{"environment": {"bar": "boo", "foo": "bar"}}`,
			expected: Environment{
				"bar": "boo",
				"foo": "bar",
			},
		},
		{
			name:     "env vars slice",
			in:       `{"environment": ["foo=bar", "bar=baz"]}`,
			expected: Environment{"bar": "baz", "foo": "bar"},
		},
		{
			name:     "env vars slice dup",
			in:       `{"environment": ["foo=bar", "bar=baz", "bar=boo"]}`,
			expected: Environment{"bar": "boo", "foo": "bar"},
		},
		{
			name:     "env vars slice of obj",
			in:       `{"environment": [{"foo":"bar"}, {"bar":"baz"}]}`,
			expected: Environment{"bar": "baz", "foo": "bar"},
		},
		{
			name:     "env vars slice of one obj",
			in:       `{"environment": [{"foo":"bar", "bar":"baz"}]}`,
			expected: Environment{"bar": "baz", "foo": "bar"},
		},
		{
			name:     "env vars slice 3=",
			in:       `{"environment": ["foo=bar=baz"]}`,
			expected: Environment{"foo": "bar=baz"},
		},
		{name: "env vars slice error - don't normalise", in: `{"environment": ["silly"]}`, expected: nil},
		{name: "env vars int error - don't normalise", in: `{"environment": [12]}`, expected: nil},
	} {
		t.Run(fmt.Sprintf("Parsing %s", tt.name), func(t *testing.T) {
			obj := map[string]any{}
			require.NoError(t, json.Unmarshal([]byte(tt.in), &obj))
			yamlBlob, err := yaml.Marshal(obj)
			require.NoError(t, err)
			env := struct {
				Environment Environment `yaml:"environment"`
			}{}

			if tt.expected == nil {
				require.ErrorContains(t, yaml.Unmarshal(yamlBlob, &env), "environment string should be of form")
			} else {
				require.NoError(t, yaml.Unmarshal(yamlBlob, &env))
				require.Equal(t, tt.expected, env.Environment)
			}
		})
	}
}
