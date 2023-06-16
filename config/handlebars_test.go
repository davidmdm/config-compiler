package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToHandleBars(t *testing.T) {
	t.Run("params", func(t *testing.T) {
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
				Input:  "<< parameters.key>>",
				Output: "{{ parameters.key}}",
			},
			{
				Input:  "<< parameters.key>>",
				Output: "{{ parameters.key}}",
			},
			{
				Input:  "<< pipeline.parameters.key >>",
				Output: "<< pipeline.parameters.key >>",
			},
			{
				Input:  "<<pipeline.parameters.key>>",
				Output: "<<pipeline.parameters.key>>",
			},
			{
				Input:  "<<random.var>>",
				Output: "<<random.var>>",
			},
		}

		for _, tc := range cases {
			t.Run(tc.Input, func(t *testing.T) {
				require.Equal(t, tc.Output, toHandlebars(tc.Input, paramExpr))
			})
		}
	})

	t.Run("pipeline parameters", func(t *testing.T) {
		cases := []struct {
			Input  string
			Output string
		}{
			{
				Input:  "<< parameters.key >>",
				Output: "<< parameters.key >>",
			},
			{
				Input:  "<< pipeline >>",
				Output: "<< pipeline >>",
			},
			{
				Input:  "<< pipeline.id >>",
				Output: "{{ pipeline.id }}",
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
				require.Equal(t, tc.Output, toHandlebars(tc.Input, pipelineParamExpr))
			})
		}
	})
}
