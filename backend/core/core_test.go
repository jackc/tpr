package core

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/tpr/backend/data"
	"github.com/stretchr/testify/require"
)

func TestEveryPrivateActionRequiresIdentity(t *testing.T) {
	c := New(nil, nil, nil)
	require.Len(t, Catalog(), 16)
	for _, a := range Catalog() {
		if a.Metadata().Public {
			continue
		}
		t.Run(a.Name(), func(t *testing.T) {
			_, err := c.InvokeJSON(context.Background(), a.Name(), []byte(`{}`))
			require.ErrorIs(t, err, ErrUnauthenticated)
		})
	}
}
func TestTypedAndDynamicCallsShareIdentity(t *testing.T) {
	c := New(nil, nil, nil)
	_, err := GetAccount.Call(context.Background(), c, Empty{})
	require.ErrorIs(t, err, ErrUnauthenticated)
	ctx := WithUser(context.Background(), &data.User{ID: pgtype.Int4{Int32: 42, Valid: true}, Name: pgtype.Text{String: "reader", Valid: true}})
	typed, err := GetAccount.Call(ctx, c, Empty{})
	require.NoError(t, err)
	dynamic, err := c.InvokeJSON(ctx, "get_account", []byte(`{}`))
	require.NoError(t, err)
	require.Equal(t, typed, dynamic)
	require.Equal(t, int32(42), typed.ID)
}
func TestPublicActionValidationWithoutStore(t *testing.T) {
	_, err := Register.Call(context.Background(), New(nil, nil, nil), Registration{Name: "reader", Password: "short"})
	var input *InputError
	require.ErrorAs(t, err, &input)
	require.Equal(t, "password must be at least 8 characters", input.Message)
}
