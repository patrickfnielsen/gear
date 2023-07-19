package deploy

import (
	"context"
	"errors"
	"os"
	"path"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/patrickfnielsen/gear/internal/gitops"
	"github.com/patrickfnielsen/gear/internal/state"
	"golang.org/x/exp/slog"
)

type RuntimeActivator struct {
	deploymentDirectory string
	state               *state.DeploymentState
}

func NewRuntimeActivator(directory string, state *state.DeploymentState) *RuntimeActivator {
	return &RuntimeActivator{
		deploymentDirectory: directory,
		state:               state,
	}
}

func (d *RuntimeActivator) DeployUpdate(ctx context.Context, bundle *gitops.Bundle) error {
	directory := path.Join(d.deploymentDirectory, bundle.Hash)

	// write all files to disk in the folder named after the commit hash
	err := d.persistBundle(bundle)
	if err != nil {
		return err
	}

	// stop all current runtimes
	for _, projectName := range d.state.DeployedServices {
		slog.Info("stopping runtime", slog.String("runtime", projectName))

		oldDirectory := path.Join(d.deploymentDirectory, d.state.CurrentHash)
		service, err := d.getComposeService(projectName, oldDirectory, []string{projectName + ".yaml"}, false)
		if err != nil {
			return errors.Join(err, errors.New("failed to get compose service"))
		}

		err = service.ComposeDown(ctx)
		if err != nil {
			return errors.Join(err, errors.New("failed to down compose service"))
		}
	}

	// startup all runtimes
	var deployed []string
	for _, dep := range bundle.Files {
		if dep.IsCustomisation {
			continue
		}

		projectName := strings.ReplaceAll(dep.FileName, ".yaml", "")
		files := []string{
			dep.FileName,
		}

		slog.Info("starting runtime", slog.String("runtime", projectName))

		if override := d.getOverride(bundle.Files, projectName); override != nil {
			slog.Info("found override for runtime", slog.String("override", override.FileName), slog.String("runtime", projectName))
			files = append(files, override.FileName)
		}

		service, err := d.getComposeService(projectName, directory, files, false)
		if err != nil {
			return errors.Join(err, errors.New("failed to get compose service"))
		}

		err = service.ComposeUp(ctx)
		if err != nil {
			return errors.Join(err, errors.New("failed to up compose service"))
		}

		slog.Info("runtime deployed", slog.String("runtime", projectName))
		deployed = append(deployed, projectName)
	}

	state, err := state.SaveDeploymentState(bundle.Hash, deployed)
	if err != nil {
		return errors.Join(err, errors.New("unable to update deployment state"))
	}

	d.state = state
	return nil
}

func (d *RuntimeActivator) persistBundle(bundle *gitops.Bundle) error {
	directory := path.Join(d.deploymentDirectory, bundle.Hash)
	err := os.MkdirAll(directory, os.ModePerm)
	if err != nil {
		return errors.Join(err, errors.New("failed to create directory for deployment"))
	}

	slog.Info("persisting bundle", slog.String("commit_hash", bundle.Hash), slog.String("directory", directory))

	for _, dep := range bundle.Files {
		fileName := path.Join(directory, dep.FileName)
		file, err := os.Create(fileName)
		if err != nil {
			return errors.Join(err, errors.New("failed to create bundle file"))
		}

		_, err = file.Write(dep.Data)
		if err != nil {
			return errors.Join(err, errors.New("failed to write bundle file"))
		}

		slog.Info("persisted bundle file", slog.String("file_name", fileName))
	}

	return nil
}

func (d *RuntimeActivator) getComposeService(name, workDir string, files []string, skipNormalization bool) (*ComposeService, error) {
	project, err := GetComposeProject(name, workDir, files, skipNormalization)
	if err != nil {
		return nil, err
	}

	logPath := path.Join(workDir, "runtime.log")
	file, err := os.Create(logPath)
	if err != nil {
		return nil, err
	}

	composeService, err := NewComposeService(command.WithCombinedStreams(file))
	if err != nil {
		return nil, err
	}
	composeService.SetProject(project)
	return composeService, nil
}

func (d *RuntimeActivator) getOverride(deployments []gitops.BundleFile, baseName string) *gitops.BundleFile {
	for _, dep := range deployments {
		if dep.IsCustomisation && strings.HasPrefix(dep.FileName, baseName) {
			return &dep
		}
	}

	return nil
}
