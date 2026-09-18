package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"

	"trefoil/x/recovery/types"
)

// ApproveRecovery is a guardian adding their vote to a pending recovery.
//
// Signed by: a guardian of the account (msg.Creator).
//
// Rules:
//  1. A recovery must be pending and not expired.
//  2. The signer must be a guardian and must not have approved already.
//  3. Once approvals reach the threshold, the waiting period starts: we stamp
//     Executeat = now + Recoverydelay. Further approvals after that are
//     rejected — the request is locked and only the account's own key can
//     cancel it.
func (k msgServer) ApproveRecovery(ctx context.Context, msg *types.MsgApproveRecovery) (*types.MsgApproveRecoveryResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}

	rec, err := k.getPendingRecovery(ctx, msg.Account)
	if err != nil {
		return nil, err
	}
	if rec.Executeat != 0 {
		return nil, types.ErrRecoveryLocked
	}

	gs, err := k.getGuardianSet(ctx, msg.Account)
	if err != nil {
		return nil, err
	}
	if !isGuardian(gs, msg.Creator) {
		return nil, types.ErrNotGuardian
	}
	for _, a := range rec.Approvals {
		if a == msg.Creator {
			return nil, types.ErrAlreadyApproved
		}
	}

	rec.Approvals = append(rec.Approvals, msg.Creator)

	if uint64(len(rec.Approvals)) >= gs.Threshold {
		params, err := k.Params.Get(ctx)
		if err != nil {
			return nil, err
		}
		rec.Executeat = blockTime(ctx) + params.Recoverydelay
	}

	if err := k.Recovery.Set(ctx, msg.Account, rec); err != nil {
		return nil, err
	}

	emit(ctx, "recovery_approved",
		"account", msg.Account,
		"guardian", msg.Creator,
		"approvals", strconv.Itoa(len(rec.Approvals)),
		"threshold", strconv.FormatUint(gs.Threshold, 10),
		"execute_at", strconv.FormatInt(rec.Executeat, 10),
	)

	return &types.MsgApproveRecoveryResponse{}, nil
}
