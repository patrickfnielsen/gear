package gitops

import (
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	"golang.org/x/exp/slog"
)

type Repository struct {
	Url    string
	Branch string
	SSHKey []byte
}

type RepositoryUpdateAvailable struct {
	Available bool
	OldHash   string
	NewHash   string
}

type BundleFile struct {
	FileName        string
	Data            []byte
	IsCustomisation bool
}

type Bundle struct {
	Hash  string
	Files []BundleFile
}

type GitOps struct {
	repo          Repository
	encryptionKey []byte
	customiseName string
	currentHash   string
}

func NewGitSync(customiseName string, currentHash string, encryptionKey []byte, repo Repository) *GitOps {
	return &GitOps{
		customiseName: customiseName,
		repo:          repo,
		currentHash:   currentHash,
		encryptionKey: encryptionKey,
	}
}

func (g *GitOps) StartSync(ctx context.Context, syncInterval int, bundleActivator func(*Bundle) error) {
	updateTicker := time.NewTicker(time.Second * time.Duration(syncInterval))

	go func(ctx context.Context) {
		defer updateTicker.Stop()

		for range updateTicker.C {
			if ctx.Err() != nil {
				return
			}

			update, err := g.CheckForUpdates()
			if err != nil {
				slog.Error("Failed to check for project updates", err, slog.String("repo", g.repo.Url))
				continue
			}

			slog.Debug(
				"Checking for project updates",
				slog.String("repo", g.repo.Url),
				slog.Bool("update_avaliable", update.Available),
				slog.String("new_hash", update.NewHash),
				slog.String("old_hash", update.OldHash),
			)

			if update.Available {
				bundle, err := g.GenerateBundle()
				if err != nil {
					slog.Error("failed to create bundle", err, slog.String("repo", g.repo.Url))
				}

				err = bundleActivator(bundle)
				if err != nil {
					slog.Error("failed to activate bundle", slog.String("error", err.Error()))
					continue
				}

				// make sure we update the current version if activation was successfull
				g.currentHash = update.NewHash
			}
		}
	}(ctx)
}

func (g *GitOps) GenerateBundle() (*Bundle, error) {
	repo, err := g.getGitRepo()
	if err != nil {
		return nil, err
	}

	wt, err := repo.Worktree()
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to get worktree"))
	}

	var bundleFiles []BundleFile
	err = util.Walk(wt.Filesystem, "", func(fileName string, fi os.FileInfo, err error) error {
		if !fi.Mode().IsRegular() || err != nil {
			return nil
		}

		// skip directories, we only want files
		if fi.IsDir() {
			return nil
		}

		// if we are in the customise directory, only get customisations for this service
		isCustomisation := false
		if strings.HasPrefix(fileName, "customise/") {
			isCustomisation = true

			if !strings.Contains(fileName, g.customiseName) {
				return nil
			}
		}

		file, err := wt.Filesystem.Open(fileName)
		defer file.Close()
		if err != nil {
			return errors.Join(err, errors.New("failed to open file"))
		}

		// handle yaml, json, and env files only
		extension := path.Ext(fileName)
		if extension == ".yaml" || extension == ".json" || extension == ".env" || extension == ".enc" {
			fileName := filepath.Base(fileName)
			bFile := BundleFile{
				FileName:        fileName,
				Data:            make([]byte, fi.Size()),
				IsCustomisation: isCustomisation,
			}

			if _, err := file.Read(bFile.Data); err != nil {
				return errors.Join(err, errors.New("failed to read file"))
			}

			// handle encrypted files
			if extension == ".enc" && g.encryptionKey != nil {
				data, err := decryptSecret(g.encryptionKey, fileName, bFile.Data)
				if err != nil {
					return errors.Join(err, errors.New("failed to decrypt secret"))
				}

				bFile.FileName = strings.ReplaceAll(fileName, ".enc", "")
				bFile.Data = data
			}

			bundleFiles = append(bundleFiles, bFile)
		}

		return nil
	})

	if err != nil {
		return nil, errors.Join(err, errors.New("failed to walk fs"))
	}

	ref, _ := repo.Head()
	return &Bundle{
		Hash:  ref.Hash().String(),
		Files: bundleFiles,
	}, nil
}

func (g *GitOps) CheckForUpdates() (*RepositoryUpdateAvailable, error) {
	head, err := g.getGitRemoteHead()
	if err != nil {
		return nil, err
	}

	update := RepositoryUpdateAvailable{
		Available: false,
		OldHash:   g.currentHash,
		NewHash:   head,
	}

	if update.OldHash != update.NewHash {
		update.Available = true
	}

	return &update, nil
}

func (g *GitOps) getGitRemoteHead() (string, error) {
	authKey, err := g.getAuthKey(g.repo.SSHKey)
	if g.repo.SSHKey != nil && err != nil {
		return "", err
	}

	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: g.repo.Url,
		URLs: []string{g.repo.Url},
	})

	list, err := remote.List(&git.ListOptions{
		Auth: authKey,
	})
	if err != nil {
		return "", errors.Join(err, errors.New("failed to list remote"))
	}

	for _, commit := range list {
		if !commit.Hash().IsZero() {
			return commit.Hash().String(), nil
		}
	}

	return "", errors.New("failed to find non zero commit")
}

func (g *GitOps) getGitRepo() (*git.Repository, error) {
	authKey, err := g.getAuthKey(g.repo.SSHKey)
	if g.repo.SSHKey != nil && err != nil {
		return nil, err
	}

	repo, err := git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:           g.repo.Url,
		ReferenceName: plumbing.NewBranchReferenceName(g.repo.Branch),
		Auth:          authKey,
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to clone"))
	}

	return repo, nil
}

func (g *GitOps) getAuthKey(sshKeyData []byte) (*ssh.PublicKeys, error) {
	authKey, err := ssh.NewPublicKeys("git", sshKeyData, "")
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to get authkey"))
	}

	return authKey, nil
}
