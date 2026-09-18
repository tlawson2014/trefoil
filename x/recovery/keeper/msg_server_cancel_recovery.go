package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	"trefoil/x/recovery/types"
)

// CancelRecovery is the veto. It is the whole reason the waiting period
// exists: if your key was stolen rather than lost, or your guardians were
// tricked, you use your still-working key to stop the recovery.
//
// Signed by: the account being recovered (msg.Creator == msg.Account).
// Guardians cannot cancel — only the owner.
func (k msgServer) CancelRecovery(ctx context.Context, msg *types.MsgCancelRecovery) (*types.MsgCancelRecoveryResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}
	if msg.Creator != msg.Account {
		return nil, types.ErrNotAccountOwner
	}

	// Use Recovery.Get directly rather than getPendingRecovery: an owner
	// should be able to clear an expired request too, and we don't want the
	// expiry side-effect to turn a valid cancel into an error.
	if _, err := k.Recovery.Get(ctx, msg.Account); err != nil {
		return nil, types.ErrNoRecoveryPending
	}
	if err := k.Recovery.Remove(ctx, msg.Account); err != nil {
		return nil, err
	}

	emit(ctx, "recovery_cancelled", "account", msg.Account)

	return &types.MsgCancelRecoveryResponse{}, nil
}
