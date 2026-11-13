package main

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/tpr/backend/core"
	"github.com/jackc/tpr/backend/data"
	"github.com/jackc/tpr/backend/jedstore"
	"github.com/stretchr/testify/require"
)

func TestResetPasswordWithJED(t *testing.T) {
	t.Setenv("TEST_ENDPOINTS", "")
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "tpr.jed")
	db, err := jedstore.Open(path)
	require.NoError(t, err)
	user := &data.User{Name: pgtype.Text{String: "reader", Valid: true}, Email: pgtype.Text{String: "reader@example.com", Valid: true}}
	require.NoError(t, core.SetPassword(user, "oldpassword"))
	_, err = data.CreateUser(ctx, db, user)
	require.NoError(t, err)
	require.NoError(t, db.Close())
	cmd := newCommand()
	cmd.Writer = io.Discard
	require.NoError(t, cmd.Run(ctx, []string{"tpr", "reset-password", "--store", "jed", "--jed-path", path, "--password", "newpassword", "reader"}))
	db, err = jedstore.Open(path)
	require.NoError(t, err)
	defer db.Close()
	got, err := data.SelectUserByName(ctx, db, "reader")
	require.NoError(t, err)
	require.True(t, core.IsPassword(got, "newpassword"))
	require.Equal(t, user.Email, got.Email)
}

func TestStorageConfigurationErrors(t *testing.T) {
	for _, args := range [][]string{{"--store", "unknown"}, {"--store", "jed"}, {"--store", "dual"}} {
		cmd := newCommand()
		cmd.Writer = io.Discard
		err := cmd.Run(context.Background(), append([]string{"tpr", "reset-password", "reader"}, args...))
		require.Error(t, err)
	}
}
