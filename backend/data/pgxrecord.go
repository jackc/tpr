package data

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgxrecord"
)

var PasswordResetsTable *pgxrecord.Table

func InitializeTables(ctx context.Context, db pgxrecord.DB) error {
	PasswordResetsTable = &pgxrecord.Table{
		Name: pgx.Identifier{"password_resets"},
	}
	err := PasswordResetsTable.LoadAllColumns(ctx, db)
	if err != nil {
		return err
	}
	PasswordResetsTable.Finalize()

	return nil
}
