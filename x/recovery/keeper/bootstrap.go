package keeper

import (
	"errors"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"trefoil/x/recovery/types"
)

// CountBootstrap records one more account created in this block via its
// first set-guardians transaction, or refuses if the block is at the cap.
//
// Only the current height's count is kept; the previous height's entry is
// dropped as we go, so this never grows.
func (k Keeper) CountBootstrap(ctx sdk.Context, max uint64) error {
	h := ctx.BlockHeight()

	n, err := k.BootstrapCount.Get(ctx, h)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		n = 0
		// New block: forget the old count.
		if h > 0 {
			if err := k.BootstrapCount.Remove(ctx, h-1); err != nil && !errors.Is(err, collections.ErrNotFound) {
				return err
			}
		}
	}
	if n >= max {
		return types.ErrTooManyNewAccounts
	}
	return k.BootstrapCount.Set(ctx, h, n+1)
}
