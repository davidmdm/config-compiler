package config_test

import (
	_ "embed"
	"testing"

	"github.com/davidmdm/config-compiler/config"
)

//go:embed test_assets/success/param_types.yml
var source []byte

func BenchmarkParamSubstitution(b *testing.B) {
	compiler := config.Compiler{}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := compiler.Compile(source, nil); err != nil {
			b.Fatal(err)
		}
	}
}
