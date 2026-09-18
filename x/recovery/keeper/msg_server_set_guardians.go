package keeper

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	"trefoil/x/recovery/types"
)

const MaxGuardians = 7

// SetGuardians lets an account choose (or replace) its guardians.
//
// Signed by: the account itself (msg.Creator).
//
// Rules, in the order they are checked:
//  1. Between 1 and 7 guardians, each a valid Trefoil address.
//  2. No duplicates, and you cannot be your own guardian.
//  3. Threshold must be a strict majority (more than half) and no more than
//     the number of guardians. For 3 guardians that means exactly 2 or 3.
//  4. You cannot change guardians while a recovery of your account is in
//     flight — otherwise whoever holds a stolen key could swap in their own
//     guardians and approve themselves.
func (k msgServer) SetGuardians(ctx context.Context, msg *types.MsgSetGuardians) (*types.MsgSetGuardiansResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}

	// 1. Count
	if len(msg.Guardians) == 0 {
		return nil, types.ErrNoGuardians
	}
	if len(msg.Guardians) > MaxGuardians {
		return nil, types.ErrTooManyGuardians
	}

	// 1 & 2. Each guardian valid, unique, and not the owner
	seen := make(map[string]struct{}, len(msg.Guardians))
	for _, g := range msg.Guardians {
		if _, err := k.addressCodec.StringToBytes(g); err != nil {
			return nil, errorsmod.Wrapf(err, "invalid guardian address %q", g)
		}
		if g == msg.Creator {
			return nil, types.ErrSelfGuardian
		}
		if _, dup := seen[g]; dup {
			return nil, errorsmod.Wrap(types.ErrDuplicateGuardian, g)
		}
		seen[g] = struct{}{}
	}

	// 3. Threshold is a strict majority: threshold*2 > n, and threshold <= n
	n := uint64(len(msg.Guardians))
	if msg.Threshold == 0 || msg.Threshold > n || msg.Threshold*2 <= n {
		return nil, errorsmod.Wrapf(types.ErrBadThreshold, "threshold %d of %d guardians", msg.Threshold, n)
	}

	// 4. Not while a recovery is pending
	_, err := k.Recovery.Get(ctx, msg.Creator)
	if err == nil {
		return nil, types.ErrRecoveryPending
	}
	if !errors.Is(err, collections.ErrNotFound) {
		return nil, err
	}

	if err := k.Guardianset.Set(ctx, msg.Creator, types.Guardianset{
		Owner:     msg.Creator,
		Threshold: msg.Threshold,
		Guardians: msg.Guardians,
	}); err != nil {
		return nil, err
	}

	emit(ctx, "guardians_set",
		"owner", msg.Creator,
		"guardians", strings.Join(msg.Guardians, ","),
		"threshold", strconv.FormatUint(msg.Threshold, 10),
	)

	return &types.MsgSetGuardiansResponse{}, nil
}
