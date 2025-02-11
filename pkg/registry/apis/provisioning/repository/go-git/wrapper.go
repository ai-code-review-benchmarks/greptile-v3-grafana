package gogit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	provisioning "github.com/grafana/grafana/pkg/apis/provisioning/v0alpha1"
	"github.com/grafana/grafana/pkg/registry/apis/provisioning/repository"
)

var (
	_ repository.Repository = (*GoGitRepo)(nil)
)

type GoGitCloneOptions struct {
	Root string // tempdir (when empty, memory??)

	// Skip intermediate commits and commit all before push
	SingleCommitBeforePush bool
}

type GoGitRepo struct {
	config *provisioning.Repository
	opts   GoGitCloneOptions

	repo *git.Repository
	tree *git.Worktree
	dir  string // file path to worktree root (necessary? should use billy)
}

// This will create a new clone every time
// As structured, it is valid for one context and should not be shared across multiple requests
func Clone(
	ctx context.Context,
	config *provisioning.Repository,
	opts GoGitCloneOptions,
	progress io.Writer, // os.Stdout
) (*GoGitRepo, error) {
	gitcfg := config.Spec.GitHub
	if gitcfg == nil {
		return nil, fmt.Errorf("missing github config")
	}
	if opts.Root == "" {
		return nil, fmt.Errorf("missing root config")
	}
	err := os.MkdirAll(opts.Root, 0700)
	if err != nil {
		return nil, err
	}
	dir, err := mkdirTempClone(opts.Root, config)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s.git", gitcfg.URL)
	repo, err := git.PlainOpen(dir)
	if err != nil {
		if !errors.Is(err, git.ErrRepositoryNotExists) {
			return nil, fmt.Errorf("error opening repository %w", err)
		}

		repo, err = git.PlainCloneContext(ctx, dir, false, &git.CloneOptions{
			Auth: &githttp.BasicAuth{
				Username: "grafana",    // this can be anything except an empty string for PAT
				Password: gitcfg.Token, // TODO... will need to get from a service!
			},
			URL:           url,
			ReferenceName: plumbing.ReferenceName(gitcfg.Branch),
			Progress:      progress,
		})
		if err != nil {
			return nil, fmt.Errorf("clone error %w", err)
		}
	}

	rcfg, err := repo.Config()
	if err != nil {
		return nil, fmt.Errorf("error readign repository config %w", err)
	}

	origin := rcfg.Remotes["origin"]
	if origin == nil {
		return nil, fmt.Errorf("missing origin remote %w", err)
	}
	if url != origin.URLs[0] {
		return nil, fmt.Errorf("unexpected remote (expected:%s, found: %s)", url, origin.URLs[0])
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(gitcfg.Branch),
		Force:  true, // clear any local changes
	})
	if err != nil {
		return nil, fmt.Errorf("unable to open branch %w", err)
	}

	return &GoGitRepo{
		config: config,
		opts:   opts,
		tree:   worktree,
		repo:   repo,
		dir:    dir,
	}, nil
}

func mkdirTempClone(root string, config *provisioning.Repository) (string, error) {
	if config.Namespace == "" {
		return "", fmt.Errorf("config is missing namespace")
	}
	if config.Name == "" {
		return "", fmt.Errorf("config is missing name")
	}
	return os.MkdirTemp(root, fmt.Sprintf("clone-%s-%s-", config.Namespace, config.Name))
}

// Remove everything from the tree
func (g *GoGitRepo) NewEmptyBranch(ctx context.Context, branch string) (int64, error) {
	err := g.tree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
		Force:  true, // clear any local changes
		Create: true,
	})
	if err != nil {
		return 0, err
	}

	count := int64(0)
	return count, util.Walk(g.tree.Filesystem, "/", func(path string, info fs.FileInfo, err error) error {
		if err != nil || strings.HasPrefix(path, "/.git") || path == "/" {
			return err
		}
		if !info.IsDir() {
			count++
			_, err = g.tree.Remove(strings.TrimLeft(path, "/"))
		}
		return err
	})
}

// Affer making changes to the worktree, push changes
func (g *GoGitRepo) Push(ctx context.Context, progress io.Writer) error {
	if g.opts.SingleCommitBeforePush {
		_, err := g.tree.Commit("exported from grafana", &git.CommitOptions{
			All: true, // Add everything that changed
		})
		if err != nil {
			return err
		}
	}

	return g.repo.PushContext(ctx, &git.PushOptions{
		Progress: progress,
		Auth: &githttp.BasicAuth{ // reuse logic from clone?
			Username: "grafana",
			Password: g.config.Spec.GitHub.Token,
		},
	})
}

// Config implements repository.Repository.
func (g *GoGitRepo) Config() *provisioning.Repository {
	return g.config
}

// ReadTree implements repository.Repository.
func (g *GoGitRepo) ReadTree(ctx context.Context, ref string) ([]repository.FileTreeEntry, error) {
	entries := make([]repository.FileTreeEntry, 0, 100)
	err := util.Walk(g.tree.Filesystem, "/", func(path string, info fs.FileInfo, err error) error {
		if err != nil || strings.HasPrefix(path, "/.git") || path == "/" {
			return err
		}
		entry := repository.FileTreeEntry{
			Path: strings.TrimLeft(path, "/"),
			Size: info.Size(),
		}
		if !info.IsDir() {
			entry.Blob = true
			// For a real instance, this will likely be based on:
			// https://github.com/go-git/go-git/blob/main/_examples/ls/main.go#L25
			entry.Hash = fmt.Sprintf("TODO/%d", info.Size()) // but not used for
		}
		entries = append(entries, entry)
		return err
	})

	if err != nil {
		return nil, err
	}
	return entries, err
}

func (g *GoGitRepo) Test(ctx context.Context) (*provisioning.TestResults, error) {
	return &provisioning.TestResults{
		Success: g.tree != nil,
	}, nil
}

// Update implements repository.Repository.
func (g *GoGitRepo) Update(ctx context.Context, path string, ref string, data []byte, message string) error {
	return g.Write(ctx, path, ref, data, message)
}

// Create implements repository.Repository.
func (g *GoGitRepo) Create(ctx context.Context, path string, ref string, data []byte, message string) error {
	return g.Write(ctx, path, ref, data, message)
}

// Write implements repository.Repository.
func (g *GoGitRepo) Write(ctx context.Context, fpath string, ref string, data []byte, message string) error {
	if err := verifyPathWithoutRef(fpath, ref); err != nil {
		return err
	}

	// For folders, just create the folder and ignore the commit
	if strings.HasSuffix(fpath, "/") {
		return g.tree.Filesystem.MkdirAll(fpath, 0750)
	}

	dir := path.Dir(fpath)
	if dir != "" {
		err := g.tree.Filesystem.MkdirAll(dir, 0750)
		if err != nil {
			return err
		}
	}

	file, err := g.tree.Filesystem.Create(fpath)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	if err != nil {
		return err
	}

	_, err = g.tree.Add(fpath)
	if err != nil {
		return err
	}

	// Skip commit for each file
	if g.opts.SingleCommitBeforePush {
		return nil
	}

	opts := &git.CommitOptions{}
	sig := repository.GetAuthorSignature(ctx)
	if sig != nil {
		opts.Author = &object.Signature{
			Name:  sig.Name,
			Email: sig.Email,
			When:  sig.When,
		}
	}
	_, err = g.tree.Commit(message, opts)
	return err
}

// Delete implements repository.Repository.
func (g *GoGitRepo) Delete(ctx context.Context, path string, ref string, message string) error {
	return g.tree.Filesystem.Remove(path) // missing slash
}

// Read implements repository.Repository.
func (g *GoGitRepo) Read(ctx context.Context, path string, ref string) (*repository.FileInfo, error) {
	stat, err := g.tree.Filesystem.Lstat(path)
	if err != nil {
		return nil, err
	}
	info := &repository.FileInfo{
		Path: path,
		Modified: &metav1.Time{
			Time: stat.ModTime(),
		},
	}
	if !stat.IsDir() {
		f, err := g.tree.Filesystem.Open(path)
		if err != nil {
			return nil, err
		}
		info.Data, err = io.ReadAll(f)
		if err != nil {
			return nil, err
		}
	}
	return info, err
}

func verifyPathWithoutRef(path string, ref string) error {
	if path == "" {
		return fmt.Errorf("expected path")
	}
	if ref != "" {
		return fmt.Errorf("ref unsupported")
	}
	return nil
}

// History implements repository.Repository.
func (g *GoGitRepo) History(ctx context.Context, path string, ref string) ([]provisioning.HistoryItem, error) {
	return nil, &apierrors.StatusError{
		ErrStatus: metav1.Status{
			Message: "history is not yet implemented",
			Code:    http.StatusNotImplemented,
		},
	}
}

// Validate implements repository.Repository.
func (g *GoGitRepo) Validate() field.ErrorList {
	return nil
}

// Webhook implements repository.Repository.
func (g *GoGitRepo) Webhook(ctx context.Context, req *http.Request) (*provisioning.WebhookResponse, error) {
	return nil, &apierrors.StatusError{
		ErrStatus: metav1.Status{
			Message: "history is not yet implemented",
			Code:    http.StatusNotImplemented,
		},
	}
}
