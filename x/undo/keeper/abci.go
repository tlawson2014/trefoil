package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"trefoil/x/undo/types"
)

// EndBlocker runs automatically at the end of every block (roughly every
// 5 seconds). Nobody signs it; the chain itself does it. It pays out every
// pending send whose undo window has closed.
//
// This is the moment a payment becomes final. Before it, the sender can
// cancel. After it, the coins are the recipient's and no one — not the
// sender, not a validator, not anyone running the chain — can reverse it.
func (k Keeper) EndBlocker(ctx context.Context) error {
	now := blockTime(ctx)

	// Collect first, act second: changing a table while walking it is a
	// classic bug.
	var due []types.Pending
	err := k.Pending.Walk(ctx, nil, func(_ uint64, p types.Pending) (bool, error) {
		if now >= p.Executeat {
			due = append(due, p)
		}
		return false, nil
	})
	if err != nil {
		return err
	}

	for _, p := range due {
		if err := k.execute(ctx, p); err != nil {
			// A failed payout must not halt the chain. Log it, leave the
			// record so it retries next block, and carry on.
			sdk.UnwrapSDKContext(ctx).Logger().Error("undo: payout failed", "id", p.Id, "err", err)
			continue
		}
	}
	return nil
}

func (k Keeper) execute(ctx context.Context, p types.Pending) error {
	to, err := k.addressCodec.StringToBytes(p.Recipient)
	if err != nil {
		return fmt.Errorf("bad recipient address: %w", err)
	}
	coins := sdk.NewCoins(p.Amount)
	// SendCoinsFromModuleToAccount creates the recipient's account if the
	// chain has never seen it, so a brand-new wallet can receive.
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, to, coins); err != nil {
		return fmt.Errorf("paying out: %w", err)
	}
	if err := k.Pending.Remove(ctx, p.Id); err != nil {
		return err
	}
	emit(ctx, "send_final",
		"id", fmt.Sprint(p.Id),
		"sender", p.Sender,
		"recipient", p.Recipient,
		"amount", coins.String(),
	)
	return nil
}
