package secretkeeper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/ini.v1"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/tracing"
	"github.com/grafana/grafana/pkg/infra/usagestats"
	"github.com/grafana/grafana/pkg/registry/apis/secret/contracts"
	encryptionmanager "github.com/grafana/grafana/pkg/registry/apis/secret/encryption/manager"
	"github.com/grafana/grafana/pkg/registry/apis/secret/secretkeeper/sqlkeeper"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/setting"
	encryptionstorage "github.com/grafana/grafana/pkg/storage/secret/encryption"
	"github.com/grafana/grafana/pkg/tests/testsuite"
)

func TestMain(m *testing.M) {
	testsuite.Run(m)
}

func Test_OSSKeeperService_GetKeepers(t *testing.T) {
	cfg := `
	[secrets_manager]
	secret_key = sdDkslslld
	encryption_provider = secretKey.v1
	available_encryption_providers = secretKey.v1
	`
	keeperService, err := setupTestService(t, cfg)
	require.NoError(t, err)

	t.Run("GetKeepers should return a map with a sql keeper", func(t *testing.T) {
		keeperMap, err := keeperService.GetKeepers()
		require.NoError(t, err)

		assert.NotNil(t, keeperMap)
		assert.Equal(t, 1, len(keeperMap))
		assert.IsType(t, &sqlkeeper.SQLKeeper{}, keeperMap[contracts.SQLKeeperType])
	})
}

func setupTestService(t *testing.T, config string) (OSSKeeperService, error) {
	raw, err := ini.Load([]byte(config))
	require.NoError(t, err)

	// Initialize data key storage and encrypted value storage with a fake db
	testDB := db.InitTestDB(t)
	cfg := &setting.Cfg{Raw: raw}
	features := featuremgmt.WithFeatures(featuremgmt.FlagGrafanaAPIServerWithExperimentalAPIs, featuremgmt.FlagSecretsManagementAppPlatform)

	dataKeyStore, err := encryptionstorage.ProvideDataKeyStorageStorage(testDB, cfg, features)
	require.NoError(t, err)

	encValueStore, err := encryptionstorage.ProvideEncryptedValueStorage(testDB, cfg, features)
	require.NoError(t, err)

	// Initialize the encryption manager
	encMgr, err := encryptionmanager.NewEncryptionManager(
		tracing.InitializeTracerForTest(),
		dataKeyStore,
		cfg,
		&usagestats.UsageStatsMock{T: t},
	)
	require.NoError(t, err)

	// Initialize the keeper service
	keeperService, err := ProvideService(tracing.InitializeTracerForTest(), encMgr, encValueStore)

	return keeperService, err
}
