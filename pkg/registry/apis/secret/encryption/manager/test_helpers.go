package manager

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/tracing"
	"github.com/grafana/grafana/pkg/infra/usagestats"
	"github.com/grafana/grafana/pkg/registry/apis/secret/encryption"
	"github.com/grafana/grafana/pkg/registry/apis/secret/encryption/cipher"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/setting"
	encryptionstorage "github.com/grafana/grafana/pkg/storage/secret/encryption"
)

func setupTestService(tb testing.TB) *EncryptionManager {
	tb.Helper()

	// Initialize data key storage with a fake db
	testDB := db.InitTestDB(tb)
	features := featuremgmt.WithFeatures(featuremgmt.FlagGrafanaAPIServerWithExperimentalAPIs, featuremgmt.FlagSecretsManagementAppPlatform)
	defaultKey := "SdlklWklckeLS"
	cfg := &setting.Cfg{
		SecretsManagement: setting.SecretsManagerSettings{
			SecretKey:          defaultKey,
			EncryptionProvider: "secretKey.v1",
			Encryption: setting.EncryptionSettings{
				DataKeysCleanupInterval: time.Nanosecond,
				DataKeysCacheTTL:        5 * time.Minute,
				Algorithm:               cipher.AesGcm,
			},
		},
	}
	store, err := encryptionstorage.ProvideDataKeyStorage(testDB, features)
	require.NoError(tb, err)

	usageStats := &usagestats.UsageStatsMock{T: tb}

	encMgr, err := ProvideEncryptionManager(
		tracing.InitializeTracerForTest(),
		store,
		cfg,
		usageStats,
		encryption.ProvideThirdPartyProviderMap(),
	)
	require.NoError(tb, err)

	return encMgr.(*EncryptionManager)
}
