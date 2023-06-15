package config_test

import (
	"bytes"
	"embed"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"compiler/config"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

//go:embed test_assets
var testAssets embed.FS

var magicSeperator = []byte(`#
# --- input above / expected below ---
#`)

func TestConfigs(t *testing.T) {
	entries, err := testAssets.ReadDir("test_assets")
	require.NoError(t, err)

	require.NoError(t, os.MkdirAll("test_output", 0o777))

	for _, file := range entries {

		if file.IsDir() {
			t.Fatalf("encountered directory within test_assets: %s", file.Name())
		}

		testName := strings.TrimSuffix(file.Name(), path.Ext(file.Name()))

		t.Run(testName, func(t *testing.T) {
			data, err := testAssets.ReadFile(path.Join("test_assets", file.Name()))
			require.NoError(t, err)

			parts := bytes.Split(data, magicSeperator)

			inputData := parts[0]
			expectedData := parts[1]

			cfg, err := config.Compile(inputData, nil)
			require.NoError(t, err)

			compiledData, err := yaml.Marshal(cfg)
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
}
