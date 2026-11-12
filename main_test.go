package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestServerCommandFlags(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test/example")
	t.Setenv("BIND_ADDRESS", "127.0.0.2")
	t.Setenv("PORT", "4000")
	t.Setenv("SMTP_SERVER", "mail.example.com")
	cmd := newCommand()
	called := false
	cmd.Commands[0].Action = func(ctx context.Context, c *cli.Command) error {
		called = true
		require.Equal(t, "postgres://test/example", c.String("database-url"))
		require.Equal(t, "127.0.0.3", c.String("bind-address"))
		require.Equal(t, "5000", c.String("port"))
		require.Equal(t, "mail.example.com", c.String("smtp-server"))
		return nil
	}
	require.NoError(t, cmd.Run(context.Background(), []string{"tpr", "s", "-a", "127.0.0.3", "-p", "5000"}))
	require.True(t, called)
}

func TestResetPasswordCommandFlags(t *testing.T) {
	cmd := newCommand()
	called := false
	cmd.Commands[1].Action = func(ctx context.Context, c *cli.Command) error {
		called = true
		require.Equal(t, "new-password", c.String("password"))
		require.Equal(t, []string{"reader"}, c.Args().Slice())
		return nil
	}
	require.NoError(t, cmd.Run(context.Background(), []string{"tpr", "reset-password", "-p", "new-password", "reader"}))
	require.True(t, called)
}

func TestResetPasswordMissingUsername(t *testing.T) {
	var output bytes.Buffer
	cmd := newCommand()
	cmd.Writer = &output
	cmd.ExitErrHandler = func(context.Context, *cli.Command, error) {}
	err := cmd.Run(context.Background(), []string{"tpr", "reset-password"})
	require.ErrorContains(t, err, "expected exactly one username")
	require.Contains(t, output.String(), "reset-password")
	require.Contains(t, output.String(), "--password")
}

func TestInventoryCommandsWithoutDatabase(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	for _, kind := range []string{"actions", "http"} {
		t.Run(kind, func(t *testing.T) {
			var output bytes.Buffer
			cmd := newCommand()
			cmd.Writer = &output
			require.NoError(t, cmd.Run(context.Background(), []string{"tpr", "inventory", kind}))
			require.Contains(t, output.String(), `"schema_version": 1`)
			if kind == "actions" {
				require.Contains(t, output.String(), `"name": "register"`)
			} else {
				require.Contains(t, output.String(), `"path": "/api/register"`)
			}
		})
	}
}
