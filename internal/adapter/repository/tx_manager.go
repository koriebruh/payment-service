package repository

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"gorm.io/gorm"

	"github.com/koriebruh/payment-service/internal/usecase/port"
)

type gormTxManager struct {
	db *gorm.DB
}

func NewGormTxManager(db *gorm.DB) port.TxManager {
	return &gormTxManager{db: db}
}

type gormTx struct {
	db *gorm.DB
}

func (t *gormTx) Commit() error {
	return t.db.Commit().Error
}

func (t *gormTx) Rollback() error {
	return t.db.Rollback().Error
}

func (m *gormTxManager) WithTx(ctx context.Context, fn func(tx port.Tx) error) error {
	ctx, span := otel.Tracer("payment-service/repository").Start(ctx, "gormTxManager.WithTx")
	defer span.End()

	txDB := m.db.WithContext(ctx).Begin()
	if txDB.Error != nil {
		return txDB.Error
	}

	tx := &gormTx{db: txDB}

	defer func() {
		if r := recover(); r != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				slog.Error("tx rollback failed during panic recover", "error", rollbackErr)
			}
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			slog.Error("tx rollback failed", "error", rollbackErr)
		}
		return err
	}

	return tx.Commit()
}

// GetGormDB is a helper to extract the gorm.DB from the port.Tx
// It falls back to the default DB if tx is nil
func GetGormDB(defaultDB *gorm.DB, tx port.Tx) *gorm.DB {
	if tx == nil {
		return defaultDB
	}
	if gTx, ok := tx.(*gormTx); ok {
		return gTx.db
	}
	return defaultDB
}
