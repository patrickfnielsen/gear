package config

import "errors"

type RepoConfig struct {
	Url                string `yaml:"url"`
	Branch             string `yaml:"branch"`
	SSHKeyFile         string `yaml:"ssh_key_file"`
	OverrideIdentifier string `yaml:"override_identifier"`
}

type DeploymentConfig struct {
	Directory string `yaml:"directory"`
}

type Config struct {
	Environment       string           `yaml:"environment"`
	SyncInterval      int              `yaml:"sync_interval"`
	EncryptionKeyFile string           `yaml:"encryption_key_file"`
	Repository        RepoConfig       `yaml:"repository"`
	Deployment        DeploymentConfig `yaml:"deployment"`
}

func (c *Config) Validate() error {
	// validate the required fields
	if c.Deployment.Directory == "" {
		return errors.New("invalid deployment directory")
	}

	if c.Repository.Branch == "" {
		return errors.New("invalid branch")
	}

	if c.Repository.SSHKeyFile == "" {
		return errors.New("invalid ssh key")
	}

	if c.Repository.Url == "" {
		return errors.New("invalid repository url")
	}

	return nil
}
