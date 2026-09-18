package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trefoil/x/recovery/types"
)

func TestBootstrapCounter(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx.WithBlockHeight(100)

	// Up to the cap is fine.
	for i := 0; i < 3; i++ {
		require.NoError(t, h.k.CountBootstrap(ctx, 3))
	}
	// The cap itself refuses.
	require.ErrorIs(t, h.k.CountBootstrap(ctx, 3), types.ErrTooManyNewAccounts)

	// Next block starts fresh, and the old block's entry is gone.
	ctx = ctx.WithBlockHeight(101)
	require.NoError(t, h.k.CountBootstrap(ctx, 3))
	_, err := h.k.BootstrapCount.Get(ctx, 100)
	require.Error(t, err, "previous block's counter should have been removed")

	n, err := h.k.BootstrapCount.Get(ctx, 101)
	require.NoError(t, err)
	require.Equal(t, uint64(1), n)
}
