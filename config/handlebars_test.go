package config

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestToHandleBars(t *testing.T) {
	cases := []struct {
		Input  string
		Output string
	}{
		{
			Input:  "<< parameters.key >>",
			Output: "{{ parameters.key }}",
		},
		{
			Input:  "<<parameters.key>>",
			Output: "{{parameters.key}}",
		},
		{
			Input:  "<< pipeline.parameters.key >>",
			Output: "{{ pipeline.parameters.key }}",
		},
		{
			Input:  "<<pipeline.parameters.key>>",
			Output: "{{pipeline.parameters.key}}",
		},
		{
			Input:  "<<random.var>>",
			Output: "<<random.var>>",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Input, func(t *testing.T) {
			require.Equal(t, tc.Output, toHandlebars(tc.Input))
		})
	}
}

func TestApplyParams(t *testing.T) {
	cfg := `image: << parameters.image >>`

	var value map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(cfg), &value))

	require.NoError(t, applyParams(&value, map[string]any{"image": "custom-image"}))
	require.Equal(t, "custom-image", value["image"])
}
