package migrator

import (
	"context"
	"fmt"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/registry/apis/secret/encryption/manager"
	legacyEncryption "github.com/grafana/grafana/pkg/services/encryption"
)

func (s simpleSecret) Rollback(
	ctx context.Context,
	secretsSrv *manager.EncryptionManager,
	encryptionSrv legacyEncryption.Internal,
	sqlStore db.DB,
	secretKey string,
) (anyFailure bool) {
	var rows []struct {
		Id     int
		Secret []byte
	}

	if err := sqlStore.WithDbSession(ctx, func(sess *db.Session) error {
		return sess.Table(s.tableName).Select(fmt.Sprintf("id, %s as secret", s.columnName)).Find(&rows)
	}); err != nil {
		logger.Warn("Could not find any secret to roll back", "table", s.tableName)
		return true
	}

	for _, row := range rows {
		if len(row.Secret) == 0 {
			continue
		}

		err := sqlStore.WithTransactionalDbSession(ctx, func(sess *db.Session) error {
			decrypted, err := secretsSrv.Decrypt(ctx, row.Secret)
			if err != nil {
				logger.Warn("Could not decrypt secret while rolling it back", "table", s.tableName, "id", row.Id, "error", err)
				return err
			}

			encrypted, err := encryptionSrv.Encrypt(ctx, decrypted, secretKey)
			if err != nil {
				logger.Warn("Could not encrypt secret while rolling it back", "table", s.tableName, "id", row.Id, "error", err)
				return err
			}

			updateSQL := fmt.Sprintf("UPDATE %s SET %s = ?, updated = ? WHERE id = ?", s.tableName, s.columnName)
			if _, err = sess.Exec(updateSQL, encrypted, nowInUTC(), row.Id); err != nil {
				logger.Warn("Could not update secret while rolling it back", "table", s.tableName, "id", row.Id, "error", err)
				return err
			}

			return nil
		})

		if err != nil {
			anyFailure = true
		}
	}

	if anyFailure {
		logger.Warn(fmt.Sprintf("Column %s from %s has been rolled back with errors", s.columnName, s.tableName))
	} else {
		logger.Info(fmt.Sprintf("Column %s from %s has been rolled back successfully", s.columnName, s.tableName))
	}

	return anyFailure
}
