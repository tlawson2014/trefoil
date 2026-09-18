package keeper

import (
	"context"
	"errors"
	"strconv"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	"trefoil/x/recovery/types"
)

// RequestRecovery opens a recovery for an account whose key is lost.
//
// Signed by: one of the account's GUARDIANS (msg.Creator), not the account.
// The person who lost their key makes a fresh wallet, reads its address to a
// guardian, and the guardian submits this. Their submission counts as the
// first approval.
//
// Why a guardian and not the new wallet itself? A brand-new address has never
// been seen by the chain, so it cannot sign a transaction yet. A guardian can.
//
// Rules:
//  1. The account must have a guardian set.
//  2. The signer must be one of its guardians.
//  3. The new address must be valid and different from the lost account.
//  4. There must not already be a live recovery for this account. An expired
//     one is cleared automatically and does not block a new request.
//
// If the threshold is 1 (single guardian), the request is immediately fully
// approved and the waiting period starts now.
func (k msgServer) RequestRecovery(ctx context.Context, msg *types.MsgRequestRecovery) (*types.MsgRequestRecoveryResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}
	if _, err := k.addressCodec.StringToBytes(msg.Account); err != nil {
		return nil, errorsmod.Wrap(err, "invalid account address")
	}

	// 3. New address sane
	if _, err := k.addressCodec.StringToBytes(msg.Newaddress); err != nil || msg.Newaddress == msg.Account {
		return nil, types.ErrBadNewAddress
	}

	// 1 & 2. Guardian set exists and signer is in it
	gs, err := k.getGuardianSet(ctx, msg.Account)
	if err != nil {
		return nil, err
	}
	if !isGuardian(gs, msg.Creator) {
		return nil, types.ErrNotGuardian
	}

	// 4. No live recovery (getPendingRecovery clears an expired one for us)
	_, err = k.getPendingRecovery(ctx, msg.Account)
	switch {
	case err == nil:
		return nil, types.ErrRecoveryPending
	case errors.Is(err, types.ErrNoRecoveryPending), errors.Is(err, types.ErrRecoveryExpired):
		// good — nothing live
	case errors.Is(err, collections.ErrNotFound):
		// good
	default:
		return nil, err
	}

	now := blockTime(ctx)
	rec := types.Recovery{
		Account:     msg.Account,
		Newaddress:  msg.Newaddress,
		Requester:   msg.Creator,
		Approvals:   []string{msg.Creator},
		Requestedat: now,
		Executeat:   0,
	}

	// Single-guardian accounts: the request is the approval.
	if uint64(len(rec.Approvals)) >= gs.Threshold {
		params, err := k.Params.Get(ctx)
		if err != nil {
			return nil, err
		}
		rec.Executeat = now + params.Recoverydelay
	}

	if err := k.Recovery.Set(ctx, msg.Account, rec); err != nil {
		return nil, err
	}

	emit(ctx, "recovery_requested",
		"account", msg.Account,
		"new_address", msg.Newaddress,
		"requester", msg.Creator,
		"execute_at", strconv.FormatInt(rec.Executeat, 10),
	)

	return &types.MsgRequestRecoveryResponse{}, nil
}
