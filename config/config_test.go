package config_test

import (
	"bytes"
	"embed"
	"fmt"
	"path"
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

			fmt.Println("---------------------")
			fmt.Println(file.Name())
			fmt.Println("---------------------")
			fmt.Println(string(compiledData))

			require.EqualValues(t, expected, actual)
		})

	}
}
