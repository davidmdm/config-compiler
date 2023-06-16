# CircleCI Config Compiler

The CircleCI Config Compiler is a Go library that allows you to compile CircleCI `config.yml` version 2.1 files to version 2.0. It provides functionality to convert and process CircleCI configuration files, enabling you to easily migrate from the newer version to the older version of the configuration format.

## Usage

To compile a `config.yml` file from version 2.1 to version 2.0, you can use the `Compile` function provided by the library. Here's an example of how to use it:

```go
package main

import (
	"fmt"
	"io"
	"log"

	"github.com/your-username/circleci-config-compiler/config"
)

func main() {
	// Read the source file
	source, err := io.ReadFile("path/to/config.yml")
	if err != nil {
		log.Fatal(err)
	}

	// Create a new compiler instance
	compiler := config.Compiler{}

	// Compile the source file
	compiledConfig, err := compiler.Compile(source, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Save the compiled config to a file
	err = io.WriteFile("path/to/compiled-config.yml", compiledConfig, 0644)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Compilation successful!")
}
```

In the above example, the `Compile` function takes the source YAML file content and an optional `pipelineParams` map, which can be used to provide parameter values for the CircleCI configuration. The function returns the compiled configuration in YAML format.

You can customize the usage according to your specific requirements and integrate it into your Go project as needed.
