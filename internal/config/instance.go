package config

import (
	"sync"
)

var (
	instance *Config
	once     sync.Once
)

// Instance returns the singleton instance of the Config
func Instance() *Config {
	if instance == nil {
		panic("Config not initialized. Call Load() first")
	}
	return instance
}

// SetInstance sets the singleton instance of the Config
// This is used primarily for testing
func SetInstance(cfg *Config) {
	instance = cfg
}

// Version returns the application version
// This is a placeholder - in a real application, this would be set during build
func (c *Config) GetVersion() string {
	return "0.1.0"
}
