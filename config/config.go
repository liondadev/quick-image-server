package config

import (
	"encoding/json"
	"fmt"
	"io"
)

// Config is the config for the application
type Config struct {
	// Users is a map of the user api key to a nice name for the user
	Users                map[string]string `json:"users"`
	DatabasePath         string            `json:"sqlite"`
	FSPath               string            `json:"storage_path"`
	BasePath             string            `json:"base_path"`
	AllowedToImportUsers []string          `json:"allowed_to_import"`
	BaseImportPath       string            `json:"base_import_path"`
}

// New returns a config with default values
func New() *Config {
	return &Config{Users: make(map[string]string)}
}

// FromReader creates a config from a reader that contains json content.
func FromReader(f io.Reader) (*Config, error) {
	cfg := New()
	if err := json.NewDecoder(f).Decode(cfg); err != nil {
		return nil, fmt.Errorf("config from reader: %w", err)
	}

	return cfg, nil
}
