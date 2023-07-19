package deploy

import (
	"context"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
	"github.com/docker/docker/client"
)

type ComposeService struct {
	api.Service
	project *types.Project
}

func NewComposeService(ops ...command.DockerCliOption) (*ComposeService, error) {
	apiClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	ops = append(ops, command.WithAPIClient(apiClient), command.WithDefaultContextStoreConfig())
	cli, err := command.NewDockerCli(ops...)
	if err != nil {
		return nil, err
	}

	cliOp := flags.NewClientOptions()
	if err := cli.Initialize(cliOp); err != nil {
		return nil, err
	}

	service := compose.NewComposeService(cli)
	return &ComposeService{service, nil}, nil
}

func (s *ComposeService) SetProject(project *types.Project) {
	s.project = project
	for i, s := range project.Services {
		s.CustomLabels = map[string]string{
			api.ProjectLabel:     project.Name,
			api.ServiceLabel:     s.Name,
			api.VersionLabel:     api.ComposeVersion,
			api.WorkingDirLabel:  project.WorkingDir,
			api.ConfigFilesLabel: strings.Join(project.ComposeFiles, ","),
			api.OneoffLabel:      "False",
		}
		project.Services[i] = s
	}
}

func (s *ComposeService) ComposeUp(ctx context.Context) error {
	return s.Up(ctx, s.project, api.UpOptions{
		Create: api.CreateOptions{
			Timeout: getComposeTimeout(),
		},
		Start: api.StartOptions{
			WaitTimeout: *getComposeTimeout(),
		},
	})
}

func (s *ComposeService) ComposeDown(ctx context.Context) error {
	return s.Down(ctx, s.project.Name, api.DownOptions{})
}

func (s *ComposeService) ComposeStart(ctx context.Context) error {
	return s.Start(ctx, s.project.Name, api.StartOptions{})
}

func (s *ComposeService) ComposeRestart(ctx context.Context) error {
	return s.Restart(ctx, s.project.Name, api.RestartOptions{})
}

func (s *ComposeService) ComposeStop(ctx context.Context) error {
	return s.Stop(ctx, s.project.Name, api.StopOptions{})
}

func (s *ComposeService) ComposeCreate(ctx context.Context) error {
	return s.Create(ctx, s.project, api.CreateOptions{})
}

func (s *ComposeService) ComposeBuild(ctx context.Context) error {
	return s.Build(ctx, s.project, api.BuildOptions{})
}

func (s *ComposeService) ComposePull(ctx context.Context) error {
	return s.Pull(ctx, s.project, api.PullOptions{})
}

func GetComposeProject(projectName, workDir string, files []string, skipNormalization bool) (*types.Project, error) {
	configFiles, err := getConfigFiles(workDir, files)
	if err != nil {
		return nil, err
	}

	details := types.ConfigDetails{
		WorkingDir:  workDir,
		ConfigFiles: configFiles,
		Environment: make(map[string]string),
	}

	projectName = strings.ToLower(projectName)
	reg, _ := regexp.Compile(`[^a-z0-9_-]+`)
	projectName = reg.ReplaceAllString(projectName, "")
	project, err := loader.Load(details, func(options *loader.Options) {
		options.SetProjectName(projectName, true)
		options.ResolvePaths = true
		options.SkipNormalization = skipNormalization
	})
	if err != nil {
		return nil, err
	}

	project.ComposeFiles = []string{configFiles[0].Filename}
	return project, nil
}

func getConfigFiles(workDir string, files []string) ([]types.ConfigFile, error) {
	var configFiles []types.ConfigFile
	for _, file := range files {
		data, err := os.ReadFile(path.Join(workDir, file))
		if err != nil {
			return configFiles, err
		}

		configFiles = append(configFiles, types.ConfigFile{Filename: file, Content: data})
	}

	return configFiles, nil
}

func getComposeTimeout() *time.Duration {
	timeout := time.Minute * time.Duration(10)
	return &timeout
}
