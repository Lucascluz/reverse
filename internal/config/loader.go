package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {

	// Validate the file path
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error loading config file: %w", err)
	}

	// Unmarshall config
	config := &Config{}
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return nil, fmt.Errorf("invalid config syntax: %w", err)
	}
	
	err = config.applyDefaults()
	if err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}
