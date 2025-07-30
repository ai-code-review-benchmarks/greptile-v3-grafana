package contracts

import (
	"context"

	common "github.com/grafana/grafana/pkg/apimachinery/apis/common/v0alpha1"
)

// SecureValue storage support
//
//go:generate mockery --name InlineSecureValueSupport --structname MockInlineSecureValueSupport --inpackage --filename inline_mock.go --with-expecter

type InlineSecureValueSupport interface {
	// Check that the request user can reference a secret in the context of a given resource (owner)
	CanReference(ctx context.Context, owner common.ObjectReference, values common.InlineSecureValues) error

	// CreateInline creates a secret that is owned by the referenced object
	// returns the name of the created secret or an error
	CreateInline(ctx context.Context, owner common.ObjectReference, value common.RawSecureValue) (string, error)

	// DeleteInline removes secrets that are owned by a referenced object
	DeleteInline(ctx context.Context, owner common.ObjectReference, name string) error
}
