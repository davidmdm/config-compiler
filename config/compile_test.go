package config_test

import (
	"bytes"
	"compiler/config"
	"embed"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidmdm/yaml"
	"github.com/stretchr/testify/require"
)

//go:embed test_assets
var testAssets embed.FS

var (
	compiledMagicSeparator = []byte(`--- # input above / compiled below`)
	errorMagicSeparator    = []byte(`--- # input above / error below`)
)

func TestConfigs(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		entries, err := testAssets.ReadDir("test_assets/success")
		require.NoError(t, err)

		require.NoError(t, os.MkdirAll("test_output", 0o777))

		for _, file := range entries {

			if file.IsDir() {
				t.Fatalf("encountered directory within test_assets/success: %s", file.Name())
			}

			testName := strings.TrimSuffix(file.Name(), path.Ext(file.Name()))

			t.Run(testName, func(t *testing.T) {
				data, err := testAssets.ReadFile(path.Join("test_assets/success", file.Name()))
				require.NoError(t, err)

				parts := bytes.Split(data, compiledMagicSeparator)

				inputData := parts[0]
				expectedData := parts[1]

				compiler := config.Compiler{}

				compiledData, err := compiler.Compile(inputData, nil)
				require.NoError(t, err)

				var (
					expected any
					actual   any
				)
				require.NoError(t, yaml.Unmarshal(expectedData, &expected))
				require.NoError(t, yaml.Unmarshal(compiledData, &actual))

				_ = os.WriteFile(filepath.Join("test_output", file.Name()), compiledData, 0o777)

				require.EqualValues(t, expected, actual)
			})
		}
	})

	t.Run("error", func(t *testing.T) {
		entries, err := testAssets.ReadDir("test_assets/error")
		require.NoError(t, err)

		for _, file := range entries {
			if file.IsDir() {
				t.Fatalf("encountered directory within test_assets/error: %s", file.Name())
			}

			testName := strings.TrimSuffix(file.Name(), path.Ext(file.Name()))

			t.Run(testName, func(t *testing.T) {
				data, err := testAssets.ReadFile(path.Join("test_assets/error", file.Name()))
				require.NoError(t, err)

				parts := bytes.Split(data, errorMagicSeparator)

				inputData := parts[0]

				var expected struct {
					Error string `yaml:"error"`
				}
				require.NoError(t, yaml.Unmarshal(parts[1], &expected))

				compiler := config.Compiler{}

				_, err = compiler.Compile(inputData, nil)
				require.EqualError(t, err, expected.Error)
			})
		}
	})
}
