package modecheck_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/grafana/grafana/pkg/storage/legacysql/modecheck"
	"github.com/grafana/grafana/pkg/storage/legacysql/modecheck/store"
)

func TestService(t *testing.T) {
	ctx := context.Background()
	mode := modecheck.ProvideModeChecker(store.ProvideStorage(nil))

	gr := schema.GroupResource{Group: "ggg", Resource: "rrr"}
	status, found := mode.Status(ctx, gr)
	require.False(t, found, "initially not found")
	require.Equal(t, modecheck.StorageStatus{
		Group:        "ggg",
		Resource:     "rrr",
		WriteLegacy:  true,
		WriteUnified: true,
		ReadUnified:  false,
		Migrated:     0,
		Migrating:    0,
		Runtime:      true,
		UpdateKey:    1,
	}, status, "should start with the right defaults")

	// Start migration
	status, err := mode.StartMigration(ctx, gr, 1)
	require.NoError(t, err)
	require.Equal(t, status.UpdateKey, int64(2), "the key increased")
	require.True(t, status.Migrating > 0, "migration is running")

	status.Migrated = time.Now().UnixMilli()
	status.Migrating = 0
	status, err = mode.Update(ctx, status)
	require.NoError(t, err)
	require.Equal(t, status.UpdateKey, int64(3), "the key increased")
	require.Equal(t, status.Migrating, int64(0), "done migrating")
	require.True(t, status.Migrated > 0, "migration is running")
}
