package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

func LoadConfig() (*Config, error) {
	// todo: handle sane defaults
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal([]byte(data), &config)
	if err != nil {
		return nil, err
	}

	return &config, config.Validate()
}
