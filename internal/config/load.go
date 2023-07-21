package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, err
	}

	// config defaults
	config := Config{
		Environment:  "PROD",
		SyncInterval: 60,
		Deployment: DeploymentConfig{
			Directory: "./deployments",
		},
	}
	err = yaml.Unmarshal([]byte(data), &config)
	if err != nil {
		return nil, err
	}

	return &config, config.Validate()
}
