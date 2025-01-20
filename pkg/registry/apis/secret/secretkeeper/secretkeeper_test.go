package secretkeeper

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/ini.v1"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/tracing"
	"github.com/grafana/grafana/pkg/infra/usagestats"
	encryptionmanager "github.com/grafana/grafana/pkg/registry/apis/secret/encryption/manager"
	encryptionprovider "github.com/grafana/grafana/pkg/services/encryption/provider"
	encryptionservice "github.com/grafana/grafana/pkg/services/encryption/service"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/kmsproviders/osskmsproviders"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/storage/secret"
	"github.com/grafana/grafana/pkg/tests/testsuite"
)

func TestMain(m *testing.M) {
	testsuite.Run(m)
}

// func Test_OSSKeeperService_GetKeeper(t *testing.T) {
// 	cfg := `
// 	[security]
// 	secret_key = sdDkslslld
// 	encryption_provider = secretKey.v1
// 	available_encryption_providers = secretKey.v1
// 	`
// 	keeperService, err := setupTestService(t, cfg)
// 	require.NoError(t, err)

// 	t.Run("GetKeeper should successfully return a sql keeper", func(t *testing.T) {
// 		keeper, err := keeperService.GetKeeper(context.Background(), keepertypes.SQLKeeperType, nil)
// 		require.NoError(t, err)
// 		assert.NotNil(t, keeper)
// 	})

// 	t.Run("GetKeeper should error when type is not sql", func(t *testing.T) {
// 		keeper, err := keeperService.GetKeeper(context.Background(), keepertypes.AWSKeeperType, nil)
// 		require.Error(t, err)
// 		assert.Nil(t, keeper)
// 	})
// }

func setupTestService(t *testing.T, config string) (OSSKeeperService, error) {
	raw, err := ini.Load([]byte(config))
	require.NoError(t, err)

	// Initialize data key storage and encrypted value storage with a fake db
	testDB := db.InitTestDB(t)
	cfg := &setting.Cfg{Raw: raw}
	features := featuremgmt.WithFeatures(featuremgmt.FlagGrafanaAPIServerWithExperimentalAPIs, featuremgmt.FlagSecretsManagementAppPlatform)

	dataKeyStore, err := secret.ProvideDataKeyStorageStorage(testDB, cfg, features)
	require.NoError(t, err)

	encValueStore, err := secret.ProvideEncryptedValueStorage(testDB, cfg, features)
	require.NoError(t, err)

	// Initialize the encryption manager
	encProvider := encryptionprovider.Provider{}
	usageStats := &usagestats.UsageStatsMock{T: t}
	encryption, err := encryptionservice.ProvideEncryptionService(tracing.InitializeTracerForTest(), encProvider, usageStats, cfg)
	require.NoError(t, err)

	encMgr, err := encryptionmanager.NewEncryptionManager(
		tracing.InitializeTracerForTest(),
		dataKeyStore,
		osskmsproviders.ProvideService(encryption, cfg, features),
		encryption,
		cfg,
		usageStats,
	)
	require.NoError(t, err)

	// Initialize the keeper service
	keeperService, err := ProvideService(encMgr, encValueStore)

	return keeperService, err
}
