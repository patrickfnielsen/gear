package state

import (
	"os"

	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
)

type DeploymentState struct {
	CurrentHash      string   `yaml:"currentHash"`
	DeployedServices []string `yaml:"deployedServices"`
}

const deploymentStateFileName = ".deployment-state.yaml"

func LoadDeploymentState() *DeploymentState {
	data, err := os.ReadFile(deploymentStateFileName)
	if err != nil {
		return &DeploymentState{}
	}

	var state DeploymentState
	err = yaml.Unmarshal([]byte(data), &state)
	if err != nil {
		slog.Warn("invalid deployment state found, starting new")
		return &DeploymentState{}
	}

	return &state
}

func SaveDeploymentState(currentHash string, deployedServices []string) (*DeploymentState, error) {
	state := DeploymentState{
		CurrentHash:      currentHash,
		DeployedServices: deployedServices,
	}

	data, err := yaml.Marshal(state)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(deploymentStateFileName, data, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return &state, nil
}
