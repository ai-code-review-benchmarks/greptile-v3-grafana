package secret

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	common "k8s.io/kube-openapi/pkg/common"

	secretV0Alpha1 "github.com/grafana/grafana/pkg/apis/secret/v0alpha1"
	"github.com/grafana/grafana/pkg/registry/apis/secret/reststorage"
	"github.com/grafana/grafana/pkg/services/apiserver/builder"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	secretstorage "github.com/grafana/grafana/pkg/storage/secret"
)

var _ builder.APIGroupBuilder = (*SecretAPIBuilder)(nil)

type SecretAPIBuilder struct {
	secureValueStorage secretstorage.SecureValueStorage
}

func NewSecretAPIBuilder(secureValueStorage secretstorage.SecureValueStorage) *SecretAPIBuilder {
	return &SecretAPIBuilder{secureValueStorage}
}

func RegisterAPIService(
	features featuremgmt.FeatureToggles,
	apiregistration builder.APIRegistrar,
	secureValueStorage secretstorage.SecureValueStorage,
) *SecretAPIBuilder {
	// Skip registration unless opting into experimental apis and the secrets management app platform flag.
	if !features.IsEnabledGlobally(featuremgmt.FlagGrafanaAPIServerWithExperimentalAPIs) ||
		!features.IsEnabledGlobally(featuremgmt.FlagSecretsManagementAppPlatform) {
		return nil
	}

	builder := NewSecretAPIBuilder(secureValueStorage)
	apiregistration.RegisterAPI(builder)
	return builder
}

// GetGroupVersion returns the tuple of `group` and `version` for the API which uniquely identifies it.
func (b *SecretAPIBuilder) GetGroupVersion() schema.GroupVersion {
	return secretV0Alpha1.SchemeGroupVersion
}

// InstallSchema is called by the `apiserver` which exposes the defined kinds.
func (b *SecretAPIBuilder) InstallSchema(scheme *runtime.Scheme) error {
	secretV0Alpha1.AddKnownTypes(scheme, secretV0Alpha1.VERSION)

	// Link this version to the internal representation.
	// This is used for server-side-apply (PATCH), and avoids the error:
	// "no kind is registered for the type"
	secretV0Alpha1.AddKnownTypes(scheme, runtime.APIVersionInternal)

	// Internal Kubernetes metadata API. Presumably to display the available APIs?
	// e.g. http://localhost:3000/apis/secret.grafana.app/v0alpha1
	metav1.AddToGroupVersion(scheme, secretV0Alpha1.SchemeGroupVersion)

	// This sets the priority in case we have multiple versions.
	// By default Kubernetes will only let you use `kubectl get <resource>` with one version.
	// In case there are multiple versions, we'd need to pass the full path with the `--raw` flag.
	if err := scheme.SetVersionPriority(secretV0Alpha1.SchemeGroupVersion); err != nil {
		return fmt.Errorf("scheme set version priority: %w", err)
	}

	return nil
}

// UpdateAPIGroupInfo is called when creating a generic API server for this group of kinds.
func (b *SecretAPIBuilder) UpdateAPIGroupInfo(apiGroupInfo *genericapiserver.APIGroupInfo, opts builder.APIGroupOptions) error {
	secureValueResource := secretV0Alpha1.SecureValuesResourceInfo
	keeperResource := secretV0Alpha1.KeeperResourceInfo

	// rest.Storage is a generic interface for RESTful storage services.
	// The constructors need to at least implement this interface, but will most likely implement
	// other interfaces that equal to different operations like `get`, `list` and so on.
	secureRestStorage := map[string]rest.Storage{
		// Default path for `securevalue`.
		// The `reststorage.SecureValueRest` struct will implement interfaces for CRUDL operations on `securevalue`.
		secureValueResource.StoragePath(): reststorage.NewSecureValueRest(b.secureValueStorage, secureValueResource),

		// The `reststorage.KeeperRest` struct will implement interfaces for CRUDL operations on `keeper`.
		keeperResource.StoragePath(): reststorage.NewKeeperRest(keeperResource),
	}

	apiGroupInfo.VersionedResourcesStorageMap[secretV0Alpha1.VERSION] = secureRestStorage
	return nil
}

// GetOpenAPIDefinitions, is this only for documentation?
func (b *SecretAPIBuilder) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions {
	return secretV0Alpha1.GetOpenAPIDefinitions
}

// GetAuthorizer: [TODO] who can create secrets? must be multi-tenant first
func (b *SecretAPIBuilder) GetAuthorizer() authorizer.Authorizer {
	// This is TBD being defined with IAM. Test

	return nil // start with the default authorizer
}

// Register additional routes with the server.
func (b *SecretAPIBuilder) GetAPIRoutes() *builder.APIRoutes {
	return nil
}

// Validate is called in `Create`, `Update` and `Delete` REST funcs, if the body calls the argument `rest.ValidateObjectFunc`.
func (b *SecretAPIBuilder) Validate(ctx context.Context, a admission.Attributes, o admission.ObjectInterfaces) error {
	return nil
}
