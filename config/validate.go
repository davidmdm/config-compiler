package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type RuleValidator []func(Config) error

func (rules RuleValidator) Validate(config Config) error {
	for _, rule := range rules {
		if err := rule(config); err != nil {
			return err
		}
	}
	return nil
}

var rules2 = RuleValidator{
	ruleMustContainJobs,
	ruleMustContainWorkflow,
	ruleSetupConfigMustBeVersion2,
	runValidateJobs,
}

var rules2_1 = RuleValidator{
	ruleMustContainJobs,
	ruleMustContainWorkflow,
	validateOrbReferences,
	ruleSetupConfigMustBeVersion2,
	ruleValidateCommands,
	runValidateJobs,
}

// Validate - the core of the validation engine. Effectively
// acts as a router based on config version and what set of validation
// rules we should use.
func Validate(yamlString string) error {
	if yamlString == "" {
		return fmt.Errorf("config string is empty")
	}

	var cfg Config
	if err := yaml.Unmarshal([]byte(yamlString), &cfg); err != nil {
		return fmt.Errorf("invalid yaml file: %w", err)
	}

	switch cfg.Version {
	case "2", "2.0":
		return rules2.Validate(cfg)
	case "2.1":
		return rules2_1.Validate(cfg)
	default:
		return fmt.Errorf("config version not supported: %q", cfg.Version)
	}
}
