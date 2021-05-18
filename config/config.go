package config

// Config for application setup
type Config interface {
	BadgerPath() (string, error)
	PushoverAPIToken() (string, error)
}
