package config

import (
	"errors"
	"fmt"
	"os"
)

const (
	// BadgerPathEnv name
	BadgerPathEnv = "BADGER_PATH"
	// PushoverAPITokenEnv name
	PushoverAPITokenEnv = "PUSHOVER_API_TOKEN"
)

var (
	// ErrEnvVariableNotSet occurs when an environment variable is not set
	ErrEnvVariableNotSet = errors.New("environment variable is not set")
)

// Env variable Config implementation
type Env struct {
}

// BadgerPath for the database directory
func (e *Env) BadgerPath() (string, error) {
	val, ok := os.LookupEnv(BadgerPathEnv)
	if !ok {
		return "", fmt.Errorf(
			"unable to get badger path from env variable %s: %w",
			BadgerPathEnv,
			ErrEnvVariableNotSet,
		)
	}

	return val, nil
}

// PushoverAPIToken getter
func (e *Env) PushoverAPIToken() (string, error) {
	val, ok := os.LookupEnv(PushoverAPITokenEnv)
	if !ok {
		return "", fmt.Errorf(
			"unable to get pushover API token from env variable %s: %w",
			PushoverAPITokenEnv,
			ErrEnvVariableNotSet,
		)
	}

	return val, nil
}
