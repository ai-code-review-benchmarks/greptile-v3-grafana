package repository

import (
	"context"
	"fmt"
	"maps"
	"slices"

	provisioning "github.com/grafana/grafana/apps/provisioning/pkg/apis/provisioning/v0alpha1"
)

//go:generate mockery --name=Extra --structname=MockExtra --inpackage --filename=extra_mock.go --with-expecter
type Extra interface {
	Type() provisioning.RepositoryType
	Build(ctx context.Context, r *provisioning.Repository) (Repository, error)
}

type Factory interface {
	Types() []provisioning.RepositoryType
	Build(ctx context.Context, r *provisioning.Repository) (Repository, error)
}

type factory struct {
	extras map[provisioning.RepositoryType]Extra
}

func NewFactory(extras []Extra) Factory {
	f := &factory{
		extras: make(map[provisioning.RepositoryType]Extra, len(extras)),
	}

	for _, e := range extras {
		f.extras[e.Type()] = e
	}

	return f
}

func (f *factory) Types() []provisioning.RepositoryType {
	return slices.Collect(maps.Keys(f.extras))
}

func (f *factory) Build(ctx context.Context, r *provisioning.Repository) (Repository, error) {
	for _, e := range f.extras {
		if e.Type() == r.Spec.Type {
			return e.Build(ctx, r)
		}
	}

	return nil, fmt.Errorf("repository type %q is not supported", r.Spec.Type)
}
