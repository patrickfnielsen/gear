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
	config, err := config.LoadConfig()
	if err != nil {
		panic("failed to load config " + err.Error())
	}

	log := logger.SetupLogger(slog.LevelDebug, config.Environment)
	log.Info("G.E.A.R (Git-Enabled Automation and Release) starting...")

	ctx := context.Background()

	log.Info("loading ssh key", slog.String("file", config.Repository.SSHKeyFile))
	sshKey, err := os.ReadFile(config.Repository.SSHKeyFile)
	if err != nil {
		panic("failed to read ssh key")
	}

	log.Info("loading deployment state")
	deploymentState := state.LoadDeploymentState()

	rintime := deploy.NewRuntimeActivator(config.Deployment.Directory, deploymentState)
	gops := gitops.NewGitOps(
		config.Repository.OverrideIdentifier,
		deploymentState.CurrentHash,
		gitops.Repository{
			Url:    config.Repository.Url,
			Branch: config.Repository.Branch,
			SSHKey: &sshKey,
		},
	)

	gops.StartUpdater(context.Background(), func(b *gitops.Bundle) error {
		log.Info("new version available", slog.String("commit_hash", b.Hash))
		return rintime.DeployUpdate(ctx, b)
	})

	quit := make(chan struct{})
	utils.MonitorSystemSignals(func(s os.Signal) {
		ctx.Done()
		close(quit)
	})

	// wait for shutdown
	<-quit
}
