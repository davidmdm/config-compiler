package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	t.Run("validate returns true for valid config files", func(t *testing.T) {
		paths, err := filepath.Glob(filepath.Join("testdata/v2_1/valid", "*.yml"))
		assert.NoError(t, err)

		for _, path := range paths {
			t.Run(strings.TrimSuffix(path, ".yml"), func(t *testing.T) {
				//nolint
				source, err := os.ReadFile(path)
				assert.NoError(t, err)
				assert.NoError(t, Validate(string(source)))
			})
		}
	})

	t.Run("validate returns false for invalid config files", func(t *testing.T) {
		paths, err := filepath.Glob(filepath.Join("testdata/v2_1/invalid", "*.yml"))
		if err != nil {
			t.Fatal(err)
		}

		expectedErrorMap := map[string]string{
			"commands-1":        "commands must have at least 1 step",
			"config1":           "config version not supported: \"\"",
			"config2":           "required key [jobs] not found",
			"config3":           "config version not supported: \"2.2\"",
			"jobs-no-steps":     "jobs must have steps",
			"macos":             "jobs require one of the follow: [macos, docker, executor] to have been defined",
			"orbs1":             "invalid yaml file: yaml: unmarshal errors:\n  line 4: cannot unmarshal !!str `node` into map[string]string",
			"setup-workflows":   "version 2.1 is required for Setup workflows",
			"steps-no-executor": "jobs require one of the follow: [macos, docker, executor] to have been defined",
		}

		for _, path := range paths {
			t.Run(strings.TrimSuffix(path, ".yml"), func(t *testing.T) {
				//nolint
				source, err := os.ReadFile(path)
				if err != nil {
					t.Fatal("error reading config file:", err)
				}
				base := func() string {
					_, filename := filepath.Split(path)
					return strings.TrimSuffix(filename, ".yml")
				}()

				assert.EqualError(t, Validate(string(source)), expectedErrorMap[base], "unexpected error for config %q", base)
				assert.Error(t, Validate(string(source)))
			})
		}
	})

	t.Run("validate returns true for valid v2 config files", func(t *testing.T) {
		paths, err := filepath.Glob(filepath.Join("testdata/v2/valid", "*.yml"))
		if err != nil {
			t.Fatal(err)
		}

		for _, path := range paths {
			t.Run(strings.TrimSuffix(path, ".yml"), func(t *testing.T) {
				//nolint
				source, err := os.ReadFile(path)
				if err != nil {
					t.Fatal("error reading config file:", err)
				}
				assert.NoError(t, Validate(string(source)))
			})
		}
	})

	t.Run("validate returns false for invalid v2 config files", func(t *testing.T) {
		paths, err := filepath.Glob(filepath.Join("testdata/v2/invalid", "*.yml"))
		if err != nil {
			t.Fatal(err)
		}

		expectedErrorMap := map[string]string{
			"setup-workflows": "version 2.1 is required for Setup workflows",
		}

		for _, path := range paths {
			t.Run(strings.TrimSuffix(path, ".yml"), func(t *testing.T) {
				//nolint
				source, err := os.ReadFile(path)
				if err != nil {
					t.Fatal("error reading config file:", err)
				}

				base := func() string {
					_, filename := filepath.Split(path)
					return strings.TrimSuffix(filename, ".yml")
				}()

				assert.EqualError(t, Validate(string(source)), expectedErrorMap[base], "unexpected error for config %q", base)
			})
		}
	})
}
