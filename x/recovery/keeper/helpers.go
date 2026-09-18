package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"trefoil/x/recovery/types"
)

// Small shared pieces used by the four message handlers and the end-blocker.

// blockTime returns the current block's timestamp in Unix seconds.
// On a blockchain "now" is the block time agreed by validators, never the
// wall clock of whichever machine happens to run the code.
func blockTime(ctx context.Context) int64 {
	return sdk.UnwrapSDKContext(ctx).BlockTime().Unix()
}

// getGuardianSet fetches an account's guardian set, or ErrNoGuardianSet if
// it never set one.
func (k Keeper) getGuardianSet(ctx context.Context, account string) (types.Guardianset, error) {
	gs, err := k.Guardianset.Get(ctx, account)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Guardianset{}, types.ErrNoGuardianSet
		}
		return types.Guardianset{}, err
	}
	return gs, nil
}

// isGuardian reports whether addr is one of the guardians in gs.
func isGuardian(gs types.Guardianset, addr string) bool {
	return contains(gs.Guardians, addr)
}

// contains reports whether s appears in list.
func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// getPendingRecovery fetches the in-flight recovery for an account.
//
// If the request has sat unapproved for longer than Requestexpiry it is
// deleted on the spot and ErrRecoveryExpired is returned, so stale requests
// can never be revived by a late approval.
func (k Keeper) getPendingRecovery(ctx context.Context, account string) (types.Recovery, error) {
	rec, err := k.Recovery.Get(ctx, account)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Recovery{}, types.ErrNoRecoveryPending
		}
		return types.Recovery{}, err
	}

	if rec.Executeat == 0 { // still collecting approvals
		params, err := k.Params.Get(ctx)
		if err != nil {
			return types.Recovery{}, err
		}
		if blockTime(ctx) > rec.Requestedat+params.Requestexpiry {
			_ = k.Recovery.Remove(ctx, account)
			return types.Recovery{}, types.ErrRecoveryExpired
		}
	}
	return rec, nil
}

// emit is a tiny helper for chain events. Events are how wallets, explorers
// and notification services find out something happened without scanning
// every account's storage.
func emit(ctx context.Context, eventType string, attrs ...string) {
	kv := make([]sdk.Attribute, 0, len(attrs)/2)
	for i := 0; i+1 < len(attrs); i += 2 {
		kv = append(kv, sdk.NewAttribute(attrs[i], attrs[i+1]))
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent(eventType, kv...))
}
