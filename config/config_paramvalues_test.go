package config

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

var yamlData = []byte(`first_name: john
last_name: doe
age: 42
nick_name: jim`)

func TestParamValues(t *testing.T) {
	type Person struct {
		FirstName string `yaml:"first_name"`
		LastName  string `yaml:"last_name"`
	}

	parent := reflect.TypeOf(Person{})

	var params ParamValues
	params.parent = parent

	require.NoError(t, yaml.Unmarshal(yamlData, &params))
	require.Len(t, params.Values, 2)

	keys := maps.Keys(params.Values)
	slices.Sort(keys)

	require.Equal(t, []string{"age", "nick_name"}, keys)
}
