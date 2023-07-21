package main

import (
	"context"
	"os"

	"github.com/patrickfnielsen/gear/internal/config"
	"github.com/patrickfnielsen/gear/internal/deploy"
	"github.com/patrickfnielsen/gear/internal/gitops"
	"github.com/patrickfnielsen/gear/internal/logger"
	"github.com/patrickfnielsen/gear/internal/state"
	"github.com/patrickfnielsen/gear/internal/utils"
	"golang.org/x/exp/slog"
)

func main() {
	ctx := context.Background()
	config, err := config.LoadConfig()
	if err != nil {
		panic("failed to load config " + err.Error())
	}

	log := logger.SetupLogger(slog.LevelDebug, config.Environment)
	log.Info("G.E.A.R (Git-Enabled Automation and Release) starting...", slog.String("environment", config.Environment))

	var sshKey []byte
	if config.Repository.SSHKeyFile != "" {
		log.Info("loading ssh key", slog.String("file", config.Repository.SSHKeyFile))
		sshKey, err = os.ReadFile(config.Repository.SSHKeyFile)
		if err != nil {
			panic("failed to read ssh key")
		}
	}

	var encryptionKey []byte
	if config.EncryptionKeyFile != "" {
		log.Info("loading encryption key", slog.String("file", config.EncryptionKeyFile))
		encryptionKey, err = os.ReadFile(config.Repository.SSHKeyFile)
		if err != nil {
			panic("failed to read encryption key")
		}
	}

	log.Info("loading deployment state")
	deploymentState := state.LoadDeploymentState()

	runtime := deploy.NewRuntimeActivator(config.Deployment.Directory, deploymentState)
	gops := gitops.NewGitSync(
		config.Repository.OverrideIdentifier,
		deploymentState.CurrentHash,
		encryptionKey,
		gitops.Repository{
			Url:    config.Repository.Url,
			Branch: config.Repository.Branch,
			SSHKey: sshKey,
		},
	)

	gops.StartSync(ctx, config.SyncInterval, func(b *gitops.Bundle) error {
		log.Info("new version available", slog.String("commit_hash", b.Hash))
		return runtime.DeployUpdate(ctx, b)
	})

	quit := make(chan struct{})
	utils.MonitorSystemSignals(func(s os.Signal) {
		ctx.Done()
		close(quit)
	})

	// wait for shutdown
	<-quit
}
