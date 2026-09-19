package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"trefoil/x/undo/types"
)

// CancelSend is the undo button.
//
// Only the original sender can press it, and only while the send is still
// pending. Once the end-blocker has paid the coins out, there is nothing to
// cancel and this fails with "no pending send with that id" — that's the
// point: after the window, a payment is final forever.
//
// The coins go straight back from holding to the sender, and the sender's
// public "cancelled" count goes up by one. That count never goes down.
func (k msgServer) CancelSend(ctx context.Context, msg *types.MsgCancelSend) (*types.MsgCancelSendResponse, error) {
	sender, err := k.addressCodec.StringToBytes(msg.Creator)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid sender address")
	}

	p, err := k.Pending.Get(ctx, msg.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, types.ErrNoSuchPending
		}
		return nil, err
	}
	if p.Sender != msg.Creator {
		return nil, types.ErrNotSender
	}

	coins := sdk.NewCoins(p.Amount)
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, sender, coins); err != nil {
		return nil, err
	}
	if err := k.Pending.Remove(ctx, msg.Id); err != nil {
		return nil, err
	}
	if err := k.bump(ctx, msg.Creator, func(r *types.Record) { r.Cancelled++ }); err != nil {
		return nil, err
	}

	emit(ctx, "send_cancelled",
		"id", fmt.Sprint(msg.Id),
		"sender", p.Sender,
		"recipient", p.Recipient,
		"amount", coins.String(),
	)
	return &types.MsgCancelSendResponse{}, nil
}
