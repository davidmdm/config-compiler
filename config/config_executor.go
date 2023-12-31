package config

import (
	"fmt"
	"regexp"

	"github.com/davidmdm/yaml"
)

type Executor struct {
	ResourceClass string   `yaml:"resource_class,omitempty"`
	Docker        []Docker `yaml:"docker,omitempty"`
	MacOS         MacOS    `yaml:"macos,omitempty"`
	Machine       Machine  `yaml:"machine,omitempty"`
}

type Machine struct {
	Image              string `yaml:"image"`
	DockerLayerCaching bool   `yaml:"docker_layer_caching,omitempty"`
}

type Docker struct {
	Image       string      `yaml:"image"`
	Name        string      `yaml:"name,omitempty"`
	EntryPoint  StringList  `yaml:"entrypoint,omitempty"`
	Command     StringList  `yaml:"command,omitempty"`
	User        string      `yaml:"user,omitempty"`
	Environment Environment `yaml:"environment,omitempty"`
	Auth        Auth        `yaml:"auth,omitempty"`     // TODO auth and AWSAuth are mutually exclusive
	AWSAuth     AWSAuth     `yaml:"aws_auth,omitempty"` // TODO auth and AWSAuth are mutually exclusive
}

// Auth contains authorization information for connecting to a Registry.
type Auth struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

type AWSAuth struct {
	AccessKey string `yaml:"aws_access_key_id,omitempty"`
	SecretKey string `yaml:"aws_secret_access_key,omitempty"`
	// TODO OIDCRoleARN is mutually exclusive to the other two keys
	OIDCRoleARN string `yaml:"oidc_role_arn,omitempty"`
}

type MacOS struct {
	XCode XCodeVersion `yaml:"xcode"`
}

type XCodeVersion string

var xCodeVersionExpression = regexp.MustCompile(`^\d(\.\d){1,2}(-\w+)?$`)

func (version *XCodeVersion) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode((*string)(version)); err != nil {
		return err
	}
	if !xCodeVersionExpression.MatchString(string(*version)) {
		return fmt.Errorf("xcode version %q does not satisfy regexp: %v", *version, xCodeVersionExpression)
	}
	return nil
}
