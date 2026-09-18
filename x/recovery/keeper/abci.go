package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"trefoil/x/recovery/types"
)

// EndBlocker runs automatically at the end of every block (roughly every
// 5 seconds). It is the part of the module nobody signs — the chain itself
// does it. Two jobs:
//
//  1. Execute recoveries whose waiting period has ended.
//  2. Drop requests that expired without reaching the threshold.
//
// Executing a recovery means: move everything the lost account holds to the
// new address, carry the guardian set and safe destinations across so the
// new account is protected from day one, then delete the old records.
//
// Prototype note: this moves the LIQUID balance. Coins that are staked stay
// staked on the old address; unstaking them is a v2 job (see README).
func (k Keeper) EndBlocker(ctx context.Context) error {
	now := blockTime(ctx)
	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}

	// Collect first, act second. Changing a map while walking it is a
	// classic bug; we avoid it by gathering the work into two lists.
	var due, expired []types.Recovery
	err = k.Recovery.Walk(ctx, nil, func(_ string, rec types.Recovery) (bool, error) {
		switch {
		case rec.Executeat != 0 && now >= rec.Executeat:
			due = append(due, rec)
		case rec.Executeat == 0 && now > rec.Requestedat+params.Requestexpiry:
			expired = append(expired, rec)
		}
		return false, nil
	})
	if err != nil {
		return err
	}

	for _, rec := range expired {
		if err := k.Recovery.Remove(ctx, rec.Account); err != nil {
			return err
		}
		emit(ctx, "recovery_expired", "account", rec.Account)
	}

	for _, rec := range due {
		if err := k.executeRecovery(ctx, rec); err != nil {
			// A failed execution must not halt the chain. Log it, leave the
			// record in place so it retries next block, and carry on.
			sdk.UnwrapSDKContext(ctx).Logger().Error("recovery execution failed",
				"account", rec.Account, "err", err)
			continue
		}
	}
	return nil
}

func (k Keeper) executeRecovery(ctx context.Context, rec types.Recovery) error {
	from, err := k.addressCodec.StringToBytes(rec.Account)
	if err != nil {
		return fmt.Errorf("bad account address: %w", err)
	}
	to, err := k.addressCodec.StringToBytes(rec.Newaddress)
	if err != nil {
		return fmt.Errorf("bad new address: %w", err)
	}

	// 1. Move the balance. SendCoins creates the destination account if it
	//    has never been seen before, which is exactly our situation.
	balance := k.bankKeeper.GetAllBalances(ctx, from)
	if !balance.IsZero() {
		if err := k.bankKeeper.SendCoins(ctx, from, to, balance); err != nil {
			return fmt.Errorf("moving balance: %w", err)
		}
	}

	// 2. Carry the guardian set across, re-owned by the new address, so the
	//    recovered person is protected immediately.
	gs, err := k.Guardianset.Get(ctx, rec.Account)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	if err == nil {
		gs.Owner = rec.Newaddress
		if err := k.Guardianset.Set(ctx, rec.Newaddress, gs); err != nil {
			return err
		}
		if err := k.Guardianset.Remove(ctx, rec.Account); err != nil {
			return err
		}
	}

	// 2b. Carry the safe destinations across too, minus the address we just
	//     recovered into (it is now "self", and self is never a destination).
	sd, err := k.Safedestinations.Get(ctx, rec.Account)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	if err == nil {
		kept := make([]string, 0, len(sd.Addresses))
		for _, a := range sd.Addresses {
			if a != rec.Newaddress {
				kept = append(kept, a)
			}
		}
		if len(kept) > 0 {
			if err := k.Safedestinations.Set(ctx, rec.Newaddress, types.Safedestinations{Owner: rec.Newaddress, Addresses: kept}); err != nil {
				return err
			}
		}
		if err := k.Safedestinations.Remove(ctx, rec.Account); err != nil {
			return err
		}
	}

	// 3. Close the recovery.
	if err := k.Recovery.Remove(ctx, rec.Account); err != nil {
		return err
	}

	emit(ctx, "recovery_executed",
		"account", rec.Account,
		"new_address", rec.Newaddress,
		"amount", balance.String(),
	)
	return nil
}
