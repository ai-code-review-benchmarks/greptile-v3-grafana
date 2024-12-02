package provisioning

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"

	"github.com/grafana/grafana/pkg/apis/provisioning/v0alpha1"
)

type helloWorldSubresource struct {
	getter        rest.Getter
	statusUpdater rest.Updater
	// for our hacky test access right now
	parent *ProvisioningAPIBuilder
	logger *slog.Logger
}

func (*helloWorldSubresource) New() runtime.Object {
	// This is added as the "ResponseType" regardless what ProducesObject() returns
	return &v0alpha1.HelloWorld{}
}

func (*helloWorldSubresource) Destroy() {}

func (*helloWorldSubresource) NamespaceScoped() bool {
	return true
}

func (*helloWorldSubresource) GetSingularName() string {
	return "HelloWorld"
}

func (*helloWorldSubresource) ProducesMIMETypes(verb string) []string {
	return []string{"application/json"}
}

func (*helloWorldSubresource) ProducesObject(verb string) any {
	return &v0alpha1.HelloWorld{}
}

func (*helloWorldSubresource) ConnectMethods() []string {
	return []string{http.MethodGet}
}

func (*helloWorldSubresource) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, false, ""
}

func (s *helloWorldSubresource) Connect(ctx context.Context, name string, opts runtime.Object, responder rest.Responder) (http.Handler, error) {
	logger := s.logger.With("repository_name", name)
	obj, err := s.getter.Get(ctx, name, &metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	repo, ok := obj.(*v0alpha1.Repository)
	if !ok {
		return nil, fmt.Errorf("expected repository, but got %t", obj)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		whom := r.URL.Query().Get("whom")
		if whom == "" {
			whom = "World"
		}
		logger := logger.With("whom", whom)

		// Exercise rendering
		renderPath := r.URL.Query().Get("render")
		if renderPath != "" {
			if !s.parent.renderer.IsAvailable(ctx) {
				responder.Error(fmt.Errorf("render not available"))
				return
			}

			ref := r.URL.Query().Get("ref")
			r, err := s.parent.GetRepository(ctx, name)
			if err != nil {
				responder.Error(err)
				return
			}

			path, err := s.parent.renderer.RenderDashboardPreview(ctx, r, renderPath, ref)
			if err != nil {
				responder.Error(err)
				return
			}

			// Show the result
			w.Write([]byte(path))

			return
		}

		newCommit := r.URL.Query().Get("commit")
		if newCommit != "" {
			repo.Status.CurrentGitCommit = newCommit
			obj, _, err := s.statusUpdater.Update(
				ctx,
				name, // resource name
				rest.DefaultUpdatedObjectInfo(obj, func(ctx context.Context, newObj, oldObj runtime.Object) (transformedNewObj runtime.Object, err error) {
					newObj.(*v0alpha1.Repository).Status.CurrentGitCommit = newCommit
					return newObj, nil
				}),
				func(ctx context.Context, obj runtime.Object) error { return nil },      // createValidation
				func(ctx context.Context, obj, old runtime.Object) error { return nil }, // updateValidation
				false,                   // forceAllowCreate
				&metav1.UpdateOptions{}, // options
			)
			if err != nil {
				responder.Error(err)
				return
			}
			repo = obj.(*v0alpha1.Repository)
		}

		logger.InfoContext(ctx, "Got a repository",
			"name", repo.Name,
			"type", repo.Spec.Type,
			"status", repo.Status,
			"local", repo.Spec.Local,
			"github", repo.Spec.GitHub,
			"s3", repo.Spec.S3)

		responder.Object(http.StatusOK, &v0alpha1.HelloWorld{Whom: whom})
	}), nil
}

var (
	_ rest.Storage              = (*helloWorldSubresource)(nil)
	_ rest.Connecter            = (*helloWorldSubresource)(nil)
	_ rest.Scoper               = (*helloWorldSubresource)(nil)
	_ rest.SingularNameProvider = (*helloWorldSubresource)(nil)
	_ rest.StorageMetadata      = (*helloWorldSubresource)(nil)
)
