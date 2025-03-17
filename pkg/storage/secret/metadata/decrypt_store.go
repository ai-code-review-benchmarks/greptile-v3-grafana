package metadata

import (
	"context"
	"errors"
	"fmt"
	"strings"

	claims "github.com/grafana/authlib/types"

	secretv0alpha1 "github.com/grafana/grafana/pkg/apis/secret/v0alpha1"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/registry/apis/secret/contracts"
	"github.com/grafana/grafana/pkg/registry/apis/secret/secretkeeper"
	"github.com/grafana/grafana/pkg/registry/apis/secret/xkube"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
)

// TODO: this should be a "decrypt" service rather, so that other services can wire and call it.
func ProvideDecryptStorage(
	db db.DB,
	features featuremgmt.FeatureToggles,
	keeperService secretkeeper.Service,
	secureValueMetadataStorage contracts.SecureValueMetadataStorage,
	allowList contracts.DecryptAllowList,
) (contracts.DecryptStorage, error) {
	if !features.IsEnabledGlobally(featuremgmt.FlagGrafanaAPIServerWithExperimentalAPIs) ||
		!features.IsEnabledGlobally(featuremgmt.FlagSecretsManagementAppPlatform) {
		return &decryptStorage{}, nil
	}

	keepers, err := keeperService.GetKeepers()
	if err != nil {
		return nil, fmt.Errorf("failed to get keepers: %w", err)
	}

	return &decryptStorage{db: db, keepers: keepers, secureValueMetadataStorage: secureValueMetadataStorage, allowList: allowList}, nil
}

// decryptStorage is the actual implementation of the decrypt storage.
type decryptStorage struct {
	// TODO: remove this and only use the storages (when making this a service), need to figure out the getKeeperConfig part.
	db db.DB

	keepers                    map[contracts.KeeperType]contracts.Keeper
	secureValueMetadataStorage contracts.SecureValueMetadataStorage
	allowList                  contracts.DecryptAllowList
}

// Decrypt decrypts a secure value from the keeper.
func (s *decryptStorage) Decrypt(ctx context.Context, namespace xkube.Namespace, name string) (secretv0alpha1.ExposedSecureValue, error) {
	authInfo, ok := claims.AuthInfoFrom(ctx)
	if !ok {
		return "", fmt.Errorf("missing auth info in context")
	}

	// The auth token will not necessarily have the permission to read the secure value metadata,
	// but we still need to do it to inspect the `decrypters` field, hence the actual `authorize`
	// function call happens after this.
	sv, err := s.secureValueMetadataStorage.Read(ctx, namespace, name)
	if err != nil {
		return "", fmt.Errorf("could not get secure value: %w", err)
	}

	authorized := s.authorize(authInfo, sv.Spec.Decrypters)
	if !authorized {
		return "", errors.New("unauthorized to decrypt")
	}

	keeperType, keeperConfig, err := getKeeperConfig(ctx, s.db, namespace.String(), sv.Spec.Keeper)
	if err != nil {
		return "", fmt.Errorf("get keeper config: %w", err)
	}

	keeper, ok := s.keepers[keeperType]
	if !ok {
		return "", fmt.Errorf("could not find keeper: %s", keeperType)
	}

	exposedValue, err := keeper.Expose(ctx, keeperConfig, namespace.String(), contracts.ExternalID(sv.Spec.Ref))
	if err != nil {
		return "", fmt.Errorf("decrypt from keeper: %w", err)
	}

	return exposedValue, nil
}

// authorize checks whether the auth info token has the right permissions to decrypt the secure value.
func (s *decryptStorage) authorize(authInfo claims.AuthInfo, svDecrypters []string) bool {
	tokenPermissions := authInfo.GetTokenPermissions()

	tokenActors := make(map[string]struct{}, 0)
	for _, permission := range tokenPermissions {
		// Will look like `secret.grafana.app/securevalues/<actor>:decrypt` for now.
		gr, verb, found := strings.Cut(permission, ":")
		if !found {
			continue
		}

		// If it isn't decrypt, then we don't care to check.
		if verb != "decrypt" {
			continue
		}

		parts := strings.Split(gr, "/")
		if len(parts) != 3 {
			continue
		}

		group, resource, actor := parts[0], parts[1], parts[2]
		if group != secretv0alpha1.GROUP || resource != secretv0alpha1.SecureValuesResourceInfo.GetName() || actor == "" {
			continue
		}

		// TEMPORARY: while we can't onboard every app into secrets, we can block them from decrypting
		// securevalues preemptively here before even reaching out to the database.
		// This check can be removed once we open the gates for any service to use secrets.
		if _, exists := s.allowList[actor]; !exists {
			continue
		}

		tokenActors[actor] = struct{}{}
	}

	// If we arrived here and the token actors is empty, it means the permissions either have an invalid format,
	// or it didn't pass the allow list, meaning no allowed decryptor.
	if len(tokenActors) == 0 {
		return false
	}

	// TEMPORARY: while we still need to mix permission and identity, we can use this
	// to decide whether the SecureValue can be decrypted or not.
	// Once we have an `actor` field in the JWT claims, we can have a properly formatted permission,
	// like `secret.grafana.app/securevalues{/<name>}:decrypt` and do regular access control eval,
	// and for the `decrypters` part here, we can just check it against the `actor` field, which at
	// that point will have a different format, depending on how the `actor` will be formatted.
	// Check whether at least one of declared token actors matches the allowed decrypters from the SecureValue.
	allowed := false

	for _, decrypter := range svDecrypters {
		if _, exists := tokenActors[decrypter]; exists {
			allowed = true
			break
		}
	}

	return allowed
}
