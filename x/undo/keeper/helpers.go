package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"trefoil/x/undo/types"
)

// Small shared pieces used by the message handlers and the end-blocker.

// blockTime returns the current block's timestamp in Unix seconds. On a
// blockchain "now" is the block time the validators agreed on, never the
// wall clock of whichever machine happens to run the code.
func blockTime(ctx context.Context) int64 {
	return sdk.UnwrapSDKContext(ctx).BlockTime().Unix()
}

// getRecord returns an address's public tally, or an empty one if the
// address has never used undo. Never an error for "not found".
func (k Keeper) getRecord(ctx context.Context, addr string) (types.Record, error) {
	r, err := k.Record.Get(ctx, addr)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Record{Index: addr}, nil
		}
		return types.Record{}, err
	}
	return r, nil
}

// bump applies a change to an address's tally and saves it.
func (k Keeper) bump(ctx context.Context, addr string, change func(*types.Record)) error {
	r, err := k.getRecord(ctx, addr)
	if err != nil {
		return err
	}
	change(&r)
	return k.Record.Set(ctx, addr, r)
}

// countOpen counts the pending sends an address currently has in flight.
// A walk over the whole table is fine for a prototype; a real deployment
// would keep a per-sender index.
func (k Keeper) countOpen(ctx context.Context, sender string) (int, error) {
	n := 0
	err := k.Pending.Walk(ctx, nil, func(_ uint64, p types.Pending) (bool, error) {
		if p.Sender == sender {
			n++
		}
		return false, nil
	})
	return n, err
}

// emit is a tiny helper for chain events. Events are how wallets and
// explorers find out something happened without scanning storage.
func emit(ctx context.Context, eventType string, attrs ...string) {
	kv := make([]sdk.Attribute, 0, len(attrs)/2)
	for i := 0; i+1 < len(attrs); i += 2 {
		kv = append(kv, sdk.NewAttribute(attrs[i], attrs[i+1]))
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent(eventType, kv...))
}
