package secret

import (
	"context"
	"fmt"

	"github.com/grafana/authlib/claims"
	secretv0alpha1 "github.com/grafana/grafana/pkg/apis/secret/v0alpha1"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/registry/apis/secret/contracts"
	"github.com/grafana/grafana/pkg/registry/apis/secret/xkube"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/sqlstore"
	"github.com/grafana/grafana/pkg/setting"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	"k8s.io/apimachinery/pkg/labels"
)

func ProvideKeeperStorage(db db.DB, cfg *setting.Cfg, features featuremgmt.FeatureToggles) (contracts.KeeperStorage, error) {
	if !features.IsEnabledGlobally(featuremgmt.FlagGrafanaAPIServerWithExperimentalAPIs) ||
		!features.IsEnabledGlobally(featuremgmt.FlagSecretsManagementAppPlatform) {
		return &keeperStorage{}, nil
	}

	return &keeperStorage{db: db}, nil
}

// keeperStorage is the actual implementation of the keeper (metadata) storage.
type keeperStorage struct {
	db db.DB
}

func (s *keeperStorage) Create(ctx context.Context, keeper *secretv0alpha1.Keeper) (*secretv0alpha1.Keeper, error) {
	authInfo, ok := claims.From(ctx)
	if !ok {
		return nil, fmt.Errorf("missing auth info in context")
	}

	row, err := toKeeperCreateRow(keeper, authInfo.GetUID())
	if err != nil {
		return nil, fmt.Errorf("failed to create row: %w", err)
	}

	err = s.db.WithTransactionalDbSession(ctx, func(sess *sqlstore.DBSession) error {
		// Validate before inserting that any `secureValues` referenced exist and do not reference other third-party keepers.
		if err := s.validateSecureValueReferences(sess, keeper); err != nil {
			return err
		}

		if _, err := sess.Insert(row); err != nil {
			return fmt.Errorf("failed to insert row: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("db failure: %w", err)
	}

	createdKeeper, err := row.toKubernetes()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to kubernetes object: %w", err)
	}

	return createdKeeper, nil
}

func (s *keeperStorage) Read(ctx context.Context, nn xkube.NameNamespace) (*secretv0alpha1.Keeper, error) {
	_, ok := claims.From(ctx)
	if !ok {
		return nil, fmt.Errorf("missing auth info in context")
	}

	row := &keeperDB{Name: nn.Name, Namespace: nn.Namespace.String()}
	err := s.db.WithDbSession(ctx, func(sess *sqlstore.DBSession) error {
		found, err := sess.Get(row)
		if err != nil {
			return fmt.Errorf("failed to get row: %w", err)
		}

		if !found {
			return contracts.ErrKeeperNotFound
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("db failure: %w", err)
	}

	keeper, err := row.toKubernetes()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to kubernetes object: %w", err)
	}

	return keeper, nil
}

func (s *keeperStorage) Update(ctx context.Context, newKeeper *secretv0alpha1.Keeper) (*secretv0alpha1.Keeper, error) {
	authInfo, ok := claims.From(ctx)
	if !ok {
		return nil, fmt.Errorf("missing auth info in context")
	}

	currentRow := &keeperDB{Name: newKeeper.Name, Namespace: newKeeper.Namespace}

	err := s.db.WithTransactionalDbSession(ctx, func(sess *sqlstore.DBSession) error {
		// Validate before updating that any `secureValues` referenced exist and do not reference other third-party keepers.
		if err := s.validateSecureValueReferences(sess, newKeeper); err != nil {
			return err
		}

		found, err := sess.Get(currentRow)
		if err != nil {
			return fmt.Errorf("failed to get row: %w", err)
		}

		if !found {
			return contracts.ErrKeeperNotFound
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("db failure: %w", err)
	}

	newRow, err := toKeeperUpdateRow(currentRow, newKeeper, authInfo.GetUID())
	if err != nil {
		return nil, fmt.Errorf("failed to map into update row: %w", err)
	}
	err = s.db.WithDbSession(ctx, func(sess *sqlstore.DBSession) error {
		if _, err := sess.Update(newRow); err != nil {
			return fmt.Errorf("failed to update row: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("db failure: %w", err)
	}

	keeper, err := newRow.toKubernetes()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to kubernetes object: %w", err)
	}
	return keeper, nil
}

func (s *keeperStorage) Delete(ctx context.Context, nn xkube.NameNamespace) error {
	_, ok := claims.From(ctx)
	if !ok {
		return fmt.Errorf("missing auth info in context")
	}

	row := &keeperDB{Name: nn.Name, Namespace: nn.Namespace.String()}
	err := s.db.WithDbSession(ctx, func(sess *sqlstore.DBSession) error {
		if _, err := sess.Delete(row); err != nil {
			return fmt.Errorf("failed to delete row: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("db failure: %w", err)
	}

	return nil
}

func (s *keeperStorage) List(ctx context.Context, namespace xkube.Namespace, options *internalversion.ListOptions) (*secretv0alpha1.KeeperList, error) {
	_, ok := claims.From(ctx)
	if !ok {
		return nil, fmt.Errorf("missing auth info in context")
	}

	labelSelector := options.LabelSelector
	if labelSelector == nil {
		labelSelector = labels.Everything()
	}

	keeperRows := make([]*keeperDB, 0)

	err := s.db.WithDbSession(ctx, func(sess *sqlstore.DBSession) error {
		cond := &keeperDB{Namespace: namespace.String()}

		if err := sess.Find(&keeperRows, cond); err != nil {
			return fmt.Errorf("failed to find rows: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("db failure: %w", err)
	}

	keepers := make([]secretv0alpha1.Keeper, 0, len(keeperRows))

	for _, row := range keeperRows {
		keeper, err := row.toKubernetes()
		if err != nil {
			return nil, fmt.Errorf("failed to convert to kubernetes object: %w", err)
		}

		if labelSelector.Matches(labels.Set(keeper.Labels)) {
			keepers = append(keepers, *keeper)
		}
	}

	return &secretv0alpha1.KeeperList{
		Items: keepers,
	}, nil
}

// validateSecureValueReferences checks that all secure values referenced by the keeper exist and are not referenced by other third-party keepers.
func (s *keeperStorage) validateSecureValueReferences(sess *sqlstore.DBSession, keeper *secretv0alpha1.Keeper) error {
	usedSecureValues := extractSecureValues(keeper)

	// No secure values are referenced, return early.
	if len(usedSecureValues) == 0 {
		return nil
	}

	secureValueCond := &secureValueDB{Namespace: keeper.Namespace}
	secureValueRows := make([]*secureValueDB, 0)

	// SELECT * FROM secret_secure_value WHERE name IN (...) AND namespace = ? FOR UPDATE;
	err := sess.Table(secureValueCond.TableName()).ForUpdate().In("name", usedSecureValues).Find(&secureValueRows, secureValueCond)
	if err != nil {
		return fmt.Errorf("check securevalues existence: %w", err)
	}

	// If not all secure values being referenced exist, return an error with the missing ones.
	if len(secureValueRows) != len(usedSecureValues) {
		// We are guaranteed that the returned `secureValueRows` are a subset of `usedSecureValues`,
		// so we don't need to check the other way around.
		missing := make(map[string]struct{}, len(usedSecureValues))
		for _, sv := range usedSecureValues {
			missing[sv] = struct{}{}
		}

		for _, svRow := range secureValueRows {
			delete(missing, svRow.Name)
		}

		return contracts.NewErrKeeperInvalidSecureValues(missing)
	}

	// If all secure values exist, we need to guarantee that the third-party keeper is not referencing another third-party,
	// it must reference only 'sql' type keepers to keep the dependency tree flat (n=1).
	keeperNames := make([]string, 0, len(secureValueRows))
	keeperSecureValues := make(map[string][]string, 0)

	for _, svRow := range secureValueRows {
		keeperNames = append(keeperNames, svRow.Keeper)
		keeperSecureValues[svRow.Keeper] = append(keeperSecureValues[svRow.Keeper], svRow.Name)
	}

	keeperCond := &keeperDB{Namespace: keeper.Namespace}
	thirdPartyKeepers := make([]*keeperDB, 0)

	// SELECT * FROM secret_keeper WHERE name IN (...) AND namespace = ? AND type != 'sql' FOR UPDATE;
	err = sess.Table(keeperCond.TableName()).ForUpdate().In("name", keeperNames).Where("type != ?", SqlKeeperType).Find(&thirdPartyKeepers, keeperCond)
	if err != nil {
		return fmt.Errorf("check keepers type: %w", err)
	}

	// Found secureValueNames that are referenced by third-party keepers.
	if len(thirdPartyKeepers) > 0 {
		invalidSecureValues := make(map[string]string, 0)

		for _, thirdPartyKeeper := range thirdPartyKeepers {
			for _, svName := range keeperSecureValues[thirdPartyKeeper.Name] {
				invalidSecureValues[svName] = thirdPartyKeeper.Name
			}
		}

		return contracts.NewErrKeeperInvalidSecureValuesReference(invalidSecureValues)
	}

	return nil
}
