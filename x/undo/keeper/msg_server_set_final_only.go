package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"

	"trefoil/x/undo/types"
)

// SetFinalOnly is the shop switch. An address that turns it on can no
// longer be sent undoable payments: anything sent to it lands immediately
// and can't be taken back. That protects a seller from "look, the payment
// is on its way" followed by a cancel.
//
// It only affects payments TO this address. The address can still send
// undoable payments to others, and can switch it back off at any time.
// Sends that were already pending when the switch is flipped are left
// alone — they finish (or are cancelled) under the rules they started with.
func (k msgServer) SetFinalOnly(ctx context.Context, msg *types.MsgSetFinalOnly) (*types.MsgSetFinalOnlyResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid address")
	}
	if err := k.bump(ctx, msg.Creator, func(r *types.Record) { r.Finalonly = msg.Finalonly }); err != nil {
		return nil, err
	}
	emit(ctx, "final_only_set", "address", msg.Creator, "finalonly", fmt.Sprint(msg.Finalonly))
	return &types.MsgSetFinalOnlyResponse{}, nil
}
