package keeper

import (
	"context"
	"errors"
	"strings"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	"trefoil/x/recovery/types"
)

const MaxSafeDestinations = 5

// SetSafeDestinations lets an account pre-register where a recovery is
// allowed to send its coins.
//
// Signed by: the account itself (msg.Creator).
//
// Why this exists: without it, two colluding guardians can recover your
// account into a wallet THEY control. With it, they can only recover into
// a wallet YOU chose in advance — a spare key in a drawer, say. Guardians
// still decide *whether* a recovery happens; they no longer decide *where*
// the money goes.
//
// Rules:
//  1. Up to 5 addresses, each valid, no duplicates, not the account itself.
//  2. An empty list is allowed and means "no restriction" (v1 behaviour).
//  3. Cannot be changed while a recovery is pending — same reasoning as
//     guardians: a thief with your key must not be able to swap in their
//     own destination mid-recovery.
func (k msgServer) SetSafeDestinations(ctx context.Context, msg *types.MsgSetSafeDestinations) (*types.MsgSetSafeDestinationsResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}

	// 1. Count, validity, uniqueness, not self
	if len(msg.Addresses) > MaxSafeDestinations {
		return nil, types.ErrTooManySafeDestinations
	}
	seen := make(map[string]struct{}, len(msg.Addresses))
	for _, a := range msg.Addresses {
		if _, err := k.addressCodec.StringToBytes(a); err != nil {
			return nil, errorsmod.Wrapf(err, "invalid safe destination %q", a)
		}
		if a == msg.Creator {
			return nil, errorsmod.Wrap(types.ErrBadSafeDestination, "an account cannot be its own safe destination")
		}
		if _, dup := seen[a]; dup {
			return nil, errorsmod.Wrapf(types.ErrBadSafeDestination, "duplicate %s", a)
		}
		seen[a] = struct{}{}
	}

	// 3. Not while a recovery is pending
	_, err := k.Recovery.Get(ctx, msg.Creator)
	if err == nil {
		return nil, types.ErrRecoveryPending
	}
	if !errors.Is(err, collections.ErrNotFound) {
		return nil, err
	}

	// 2. Empty list clears the restriction
	if len(msg.Addresses) == 0 {
		if err := k.Safedestinations.Remove(ctx, msg.Creator); err != nil && !errors.Is(err, collections.ErrNotFound) {
			return nil, err
		}
		emit(ctx, "safe_destinations_cleared", "owner", msg.Creator)
		return &types.MsgSetSafeDestinationsResponse{}, nil
	}

	if err := k.Safedestinations.Set(ctx, msg.Creator, types.Safedestinations{
		Owner:     msg.Creator,
		Addresses: msg.Addresses,
	}); err != nil {
		return nil, err
	}

	emit(ctx, "safe_destinations_set",
		"owner", msg.Creator,
		"addresses", strings.Join(msg.Addresses, ","),
	)

	return &types.MsgSetSafeDestinationsResponse{}, nil
}
