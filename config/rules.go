package config

import (
	"errors"
	"fmt"
	"reflect"
)

func ruleMustContainJobs(c Config) error {
	if len(c.Jobs) == 0 {
		return fmt.Errorf("required key [jobs] not found")
	}
	return nil
}

func ruleMustContainWorkflow(c Config) error {
	if c.Workflows == nil {
		return fmt.Errorf("required key [workflows] not found")
	}
	return nil
}

func ruleSetupConfigMustBeVersion2(c Config) error {
	if c.Setup && c.Version != "2.1" {
		return fmt.Errorf("version 2.1 is required for Setup workflows")
	}
	return nil
}

func ruleValidateCommands(c Config) error {
	for _, com := range c.Commands {
		if len(com.Steps) == 0 {
			return fmt.Errorf("commands must have at least 1 step")
		}
	}
	return nil
}

func runValidateJobs(c Config) error {
	for _, j := range c.Jobs {
		// validate that all the jobs in the config have at least
		// 1 step defined in the job
		if len(j.Steps) <= 0 {
			return fmt.Errorf("jobs must have steps")
		}

		if j.Executor.Name != "" {
			continue // TODO check executor reference exists?
		}

		if !reflect.ValueOf(j.InlineExecutor).IsZero() {
			continue
		}

		return errors.New("jobs require one of the follow: [macos, docker, executor] to have been defined")
	}
	return nil
}

// TODO: this has to make calls to the orb-service database to validate
// that the orb exists.
func validateOrbReferences(c Config) error {
	for key, orbValue := range c.Orbs {
		fmt.Printf("%s : %s \n", key, orbValue)
	}
	return nil
}
