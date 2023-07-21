package config_test

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"

	"github.com/davidmdm/config-compiler/config"

	"github.com/davidmdm/yaml"
	"github.com/stretchr/testify/require"
)

//go:embed test_assets
var testAssets embed.FS

const (
	tempPath         = "test_output/temp.yaml"
	tempCompiledPath = "test_output/temp_compiled.yaml"
)

var e2e = os.Getenv("E2E") == "true"

func init() {
	if err := exec.Command("which", "circleci").Run(); err != nil {
		fmt.Println(err)
		e2e = false
	}
}

var (
	compiledMagicSeparator = []byte(`--- # input above / compiled below`)
	errorMagicSeparator    = []byte(`--- # input above / error below`)
	paramSeparator         = []byte(`--- # pipeline parameters`)
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

			t.Run(file.Name(), func(t *testing.T) {
				data, err := testAssets.ReadFile(path.Join("test_assets/success", file.Name()))
				require.NoError(t, err)

				parts := bytes.Split(data, compiledMagicSeparator)

				inputData := parts[0]

				inputData, paramData, _ := bytes.Cut(inputData, paramSeparator)

				var params map[string]any
				require.NoError(t, yaml.Unmarshal(paramData, &params))

				expectedData := parts[1]

				compiler := config.Compiler{}

				compiledData, err := compiler.Compile(inputData, params)
				require.NoError(t, err)

				var (
					expected any
					actual   any
				)
				require.NoError(t, yaml.Unmarshal(expectedData, &expected))
				require.NoError(t, yaml.Unmarshal(compiledData, &actual))

				_ = os.WriteFile(filepath.Join("test_output", file.Name()), compiledData, 0o777)

				require.EqualValues(t, expected, actual)

				if !e2e {
					return
				}

				require.NoError(t, os.WriteFile(tempPath, inputData, 0o777))

				out, err := exec.Command("circleci", "--skip-update-check", "config", "process", tempPath).CombinedOutput()
				require.NoError(t, err, string(out))

				require.NoError(t, os.WriteFile(tempCompiledPath, out, 0o777))

				var cci any
				require.NoError(t, yaml.Unmarshal(out, &cci))

				m := cci.(map[string]any)["workflows"].(map[string]any)
				delete(m, "version")

				require.EqualValues(t, cci, actual)
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

			t.Run(file.Name(), func(t *testing.T) {
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
