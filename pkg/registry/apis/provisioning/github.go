package provisioning

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"

	provisioning "github.com/grafana/grafana/pkg/apis/provisioning/v0alpha1"
)

type githubRepository struct {
	logger       *slog.Logger
	config       *provisioning.Repository
	githubClient *github.Client
}

func newGithubRepository(ctx context.Context, config *provisioning.Repository) *githubRepository {
	tokenSrc := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.Spec.GitHub.Token},
	)
	tokenClient := oauth2.NewClient(ctx, tokenSrc)
	githubClient := github.NewClient(tokenClient)

	return &githubRepository{
		config:       config,
		githubClient: githubClient,
		logger:       slog.Default().With("logger", "github-repository"),
	}
}

func (r *githubRepository) Config() *provisioning.Repository {
	return r.config
}

// Validate implements provisioning.Repository.
func (r *githubRepository) Validate() (list field.ErrorList) {
	gh := r.config.Spec.GitHub
	if gh == nil {
		list = append(list, field.Required(field.NewPath("spec", "github"), "a github config is required"))
		return list
	}
	if gh.Owner == "" {
		list = append(list, field.Required(field.NewPath("spec", "github", "owner"), "a github repo owner is required"))
	}
	if gh.Repository == "" {
		list = append(list, field.Required(field.NewPath("spec", "github", "repository"), "a github repo name is required"))
	}
	if gh.Branch == "" {
		list = append(list, field.Required(field.NewPath("spec", "github", "branch"), "a github branch is required"))
	}
	if gh.Token == "" {
		list = append(list, field.Required(field.NewPath("spec", "github", "token"), "a github access token is required"))
	}
	if gh.GenerateDashboardPreviews && !gh.BranchWorkflow {
		list = append(list, field.Forbidden(field.NewPath("spec", "github", "token"), "to generate dashboard previews, you must activate the branch workflow"))
	}
	if gh.WebhookURL == "" {
		list = append(list, field.Required(field.NewPath("spec", "github", "webhookURL"), "a webhook URL is required"))
	}
	if gh.WebhookSecret == "" {
		list = append(list, field.Required(field.NewPath("spec", "github", "webhookSecret"), "a webhook secret is required"))
	}

	_, err := url.Parse(gh.WebhookURL)
	if err != nil {
		list = append(list, field.Invalid(field.NewPath("spec", "github", "webhookURL"), gh.WebhookURL, "invalid URL"))
	}

	return list
}

// Test implements provisioning.Repository.
func (r *githubRepository) Test(ctx context.Context) error {
	return &apierrors.StatusError{
		ErrStatus: metav1.Status{
			Message: "test is not yet implemented",
			Code:    http.StatusNotImplemented,
		},
	}
}

// ReadResource implements provisioning.Repository.
func (r *githubRepository) Read(ctx context.Context, filePath string, ref string) (*FileInfo, error) {
	if ref == "" {
		ref = r.config.Spec.GitHub.Branch
	}

	content, _, _, err := r.githubClient.Repositories.GetContents(ctx, r.config.Spec.GitHub.Owner, r.config.Spec.GitHub.Repository, filePath, &github.RepositoryContentGetOptions{
		Ref: ref,
	})
	if err != nil {
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) && ghErr.Response.StatusCode == http.StatusNotFound {
			return nil, &apierrors.StatusError{
				ErrStatus: metav1.Status{
					Message: fmt.Sprintf("file not found; path=%s ref=%s", filePath, ref),
					Code:    http.StatusNotFound,
				},
			}
		}

		return nil, fmt.Errorf("get contents: %w", err)
	}

	data, err := content.GetContent()
	if err != nil {
		return nil, fmt.Errorf("get content: %w", err)
	}
	return &FileInfo{
		Path: filePath,
		Ref:  ref,
		Data: []byte(data),
		Hash: content.GetSHA(),
	}, nil
}

func (r *githubRepository) Create(ctx context.Context, path string, ref string, data []byte, comment string) error {
	if _, _, err := r.githubClient.Repositories.CreateFile(ctx, r.config.Spec.GitHub.Owner, r.config.Spec.GitHub.Repository, path, &github.RepositoryContentFileOptions{
		Message: github.String(comment),
		Content: data,
		Branch:  github.String(r.config.Spec.GitHub.Branch),
	}); err != nil {
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) && ghErr.Response.StatusCode == http.StatusUnprocessableEntity {
			return &apierrors.StatusError{
				ErrStatus: metav1.Status{
					Message: "file already exists",
					Code:    http.StatusConflict,
				},
			}
		}

		return err
	}

	return nil
}

func (r *githubRepository) Update(ctx context.Context, path string, ref string, data []byte, comment string) error {
	file, _, _, err := r.githubClient.Repositories.GetContents(ctx, r.config.Spec.GitHub.Owner, r.config.Spec.GitHub.Repository, path, &github.RepositoryContentGetOptions{
		Ref: r.config.Spec.GitHub.Branch,
	})

	if err != nil {
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) && ghErr.Response.StatusCode == http.StatusNotFound {
			return &apierrors.StatusError{
				ErrStatus: metav1.Status{
					Message: "file not found",
					Code:    http.StatusNotFound,
				},
			}
		}

		return fmt.Errorf("get content before file update: %w", err)
	}

	if _, _, err = r.githubClient.Repositories.UpdateFile(ctx, r.config.Spec.GitHub.Owner, r.config.Spec.GitHub.Repository, path, &github.RepositoryContentFileOptions{
		Message: github.String(comment),
		Content: data,
		SHA:     file.SHA,
		Branch:  github.String(r.config.Spec.GitHub.Branch),
	}); err != nil {
		return fmt.Errorf("update file: %w", err)
	}

	return nil
}

func (r *githubRepository) Delete(ctx context.Context, path string, ref string, comment string) error {
	file, _, _, err := r.githubClient.Repositories.GetContents(ctx, r.config.Spec.GitHub.Owner, r.config.Spec.GitHub.Repository, path, &github.RepositoryContentGetOptions{
		Ref: r.config.Spec.GitHub.Branch,
	})
	if err != nil {
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) && ghErr.Response.StatusCode == http.StatusNotFound {
			return &apierrors.StatusError{
				ErrStatus: metav1.Status{
					Message: "file not found",
					Code:    http.StatusNotFound,
				},
			}
		}
	}

	if _, _, err = r.githubClient.Repositories.DeleteFile(ctx, r.config.Spec.GitHub.Owner, r.config.Spec.GitHub.Repository, path, &github.RepositoryContentFileOptions{
		Message: github.String(comment),
		Branch:  github.String(r.config.Spec.GitHub.Branch),
		SHA:     file.SHA,
	}); err != nil {
		return err
	}

	return nil
}

// Webhook implements provisioning.Repository.
func (r *githubRepository) Webhook(responder rest.Responder) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		payload, err := github.ValidatePayload(req, []byte(r.config.Spec.GitHub.WebhookSecret))
		if err != nil {
			responder.Error(apierrors.NewUnauthorized("invalid signature"))
			return
		}

		eventType := github.WebHookType(req)
		event, err := github.ParseWebHook(github.WebHookType(req), payload)
		if err != nil {
			responder.Error(apierrors.NewBadRequest("invalid payload"))
			return
		}

		switch event := event.(type) {
		case *github.PushEvent:
			if err := r.onPushEvent(req.Context(), event); err != nil {
				responder.Error(err)
				return
			}

			responder.Object(200, &metav1.Status{
				Message: "event processed",
				Code:    http.StatusOK,
			})
		case *github.PingEvent:
			responder.Object(200, &metav1.Status{
				Message: "ping received",
				Code:    http.StatusOK,
			})
		default:
			responder.Error(apierrors.NewBadRequest("unsupported event type: " + eventType))
		}
	}
}

func (r *githubRepository) onPushEvent(ctx context.Context, event *github.PushEvent) error {
	if event.GetRepo() == nil {
		return fmt.Errorf("missing repository in push event")
	}

	if event.GetRepo().GetFullName() != fmt.Sprintf("%s/%s", r.config.Spec.GitHub.Owner, r.config.Spec.GitHub.Repository) {
		return fmt.Errorf("repository mismatch")
	}

	// Skip silently if the event is not for the main/master branch
	// as we cannot configure the webhook to only publish events for the main branch
	if event.GetRef() != fmt.Sprintf("refs/heads/%s", r.config.Spec.GitHub.Branch) {
		return nil
	}

	beforeRef := event.GetBefore()

	for _, commit := range event.Commits {
		r.logger.Info("process commit", "commit", commit.GetID(), "message", commit.GetMessage())

		for _, file := range commit.Added {
			info, err := r.Read(ctx, file, commit.GetID())
			if err != nil {
				return fmt.Errorf("read added resource: %w", err)
			}

			r.logger.Info("added file", "file", file, "resource", string(info.Data), "commit", commit.GetID())
		}

		for _, file := range commit.Modified {
			info, err := r.Read(ctx, file, commit.GetID())
			if err != nil {
				return fmt.Errorf("read modified resource: %w", err)
			}

			r.logger.Info("modified file", "file", file, "resource", string(info.Data), "commit", commit.GetID())
		}

		for _, file := range commit.Removed {
			info, err := r.Read(ctx, file, beforeRef)
			if err != nil {
				return fmt.Errorf("read removed resource: %w", err)
			}

			r.logger.Info("removed file", "file", file, "resource", string(info.Data), "commit", commit.GetID())
		}

		beforeRef = commit.GetID()
	}

	return nil
}

func (r *githubRepository) createWebhook(ctx context.Context) error {
	hook := &github.Hook{
		Config: &github.HookConfig{
			URL:         github.String(r.config.Spec.GitHub.WebhookURL),
			ContentType: github.String("json"),
			Secret:      github.String(r.config.Spec.GitHub.WebhookSecret),
		},
		Events: []string{"push"},
		Active: github.Bool(true),
	}

	if _, _, err := r.githubClient.Repositories.CreateHook(ctx, r.config.Spec.GitHub.Owner, r.config.Spec.GitHub.Repository, hook); err != nil {
		return err
	}

	r.logger.Info("webhook created", "url", r.config.Spec.GitHub.WebhookURL)

	return nil
}

func (r *githubRepository) updateWebhook(ctx context.Context, oldRepo *githubRepository) (UndoFunc, error) {
	owner := r.config.Spec.GitHub.Owner
	repoName := r.config.Spec.GitHub.Repository

	hooks, _, err := r.githubClient.Repositories.ListHooks(ctx, owner, repoName, nil)
	if err != nil {
		return nil, fmt.Errorf("list existing webhooks: %w", err)
	}

	newCfg := r.config.Spec.GitHub
	oldCfg := oldRepo.Config().Spec.GitHub

	switch {
	case newCfg.WebhookURL != oldCfg.WebhookURL:
		// In this case we cannot find out out which webhook to update, so we delete the old one and create a new one
		if err := r.createWebhook(ctx); err != nil {
			return nil, fmt.Errorf("create new webhook: %w", err)
		}

		undoFunc := UndoFunc(func(ctx context.Context) error {
			if err := r.deleteWebhook(ctx); err != nil {
				return fmt.Errorf("revert create new webhook: %w", err)
			}

			r.logger.Info("create new webhook reverted", "url", newCfg.WebhookURL)

			return nil
		})

		if err := oldRepo.deleteWebhook(ctx); err != nil {
			return undoFunc, fmt.Errorf("delete old webhook: %w", err)
		}

		undoFunc = undoFunc.Chain(ctx, func(ctx context.Context) error {
			if err := oldRepo.createWebhook(ctx); err != nil {
				return fmt.Errorf("revert delete old webhook: %w", err)
			}

			r.logger.Info("delete old webhook reverted", "url", oldCfg.WebhookURL)

			return nil
		})

		return undoFunc, nil
	case newCfg.WebhookSecret != oldCfg.WebhookSecret:
		for _, hook := range hooks {
			if *hook.Config.URL == oldCfg.WebhookURL {
				hook.Config.Secret = github.String(newCfg.WebhookSecret)
				_, _, err := r.githubClient.Repositories.EditHook(ctx, owner, repoName, *hook.ID, hook)
				if err != nil {
					return nil, fmt.Errorf("update webhook secret: %w", err)
				}

				r.logger.Info("webhook secret updated", "url", newCfg.WebhookURL)

				return func(ctx context.Context) error {
					hook.Config.Secret = github.String(oldCfg.WebhookSecret)
					if _, _, err := r.githubClient.Repositories.EditHook(ctx, owner, repoName, *hook.ID, hook); err != nil {
						return fmt.Errorf("revert webhook secret: %w", err)
					}

					r.logger.Info("webhook secret reverted", "url", oldCfg.WebhookURL)
					return nil
				}, nil
			}
		}

		return nil, errors.New("webhook not found")
	default:
		return nil, nil
	}
}

func (r *githubRepository) deleteWebhook(ctx context.Context) error {
	owner := r.config.Spec.GitHub.Owner
	name := r.config.Spec.GitHub.Repository

	hooks, _, err := r.githubClient.Repositories.ListHooks(ctx, owner, name, nil)
	if err != nil {
		return fmt.Errorf("list existing webhooks: %w", err)
	}

	for _, hook := range hooks {
		if *hook.Config.URL == r.config.Spec.GitHub.WebhookURL {
			if _, err := r.githubClient.Repositories.DeleteHook(ctx, owner, name, *hook.ID); err != nil {
				return fmt.Errorf("delete webhook: %w", err)
			}
		}
	}

	r.logger.Info("webhook deleted", "url", r.config.Spec.GitHub.WebhookURL)
	return nil
}

func (r *githubRepository) AfterCreate(ctx context.Context) error {
	return r.createWebhook(ctx)
}

func (r *githubRepository) BeginUpdate(ctx context.Context, old Repository) (UndoFunc, error) {
	oldGitRepo, ok := old.(*githubRepository)
	if !ok {
		return nil, fmt.Errorf("old repository is not a github repository")
	}

	return r.updateWebhook(ctx, oldGitRepo)
}

func (r *githubRepository) AfterDelete(ctx context.Context) error {
	return r.deleteWebhook(ctx)
}
