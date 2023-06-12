package config

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
	Auth        Auth        `yaml:"auth,omitempty"`
	AWSAuth     AWSAuth     `yaml:"aws_auth,omitempty"`
}

type Auth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type AWSAuth struct {
	AccessKey   string `yaml:"aws_access_key_id,omitempty"`
	SecretKey   string `yaml:"aws_secret_access_key,omitempty"`
	OIDCRoleARN string `yaml:"oidc_role_arn,omitempty"`
}

type MacOS struct {
	XCode string `yaml:"xcode"`
}
