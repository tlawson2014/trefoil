package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"trefoil/x/undo/types"
)

// DelayedSend is Trefoil's send. Every payment on this chain goes through
// it, and every payment has an undo window unless the recipient has opted
// out with SetFinalOnly.
//
// What happens:
//
//  1. Basic checks: real addresses, not to yourself, a positive amount.
//  2. If the recipient is final-only, the sender must have said window 0
//     (the wallet asks them to confirm), and the coins move immediately.
//     Nothing is pending; nothing can be undone.
//  3. Otherwise the window is checked against the chain's min/max, the
//     coins move into the module's holding account, and a Pending record
//     is written with the time they become the recipient's. The sender's
//     public tally goes up by one.
//
// The recipient sees the pending send; the coins are not theirs until the
// end-blocker pays them out. Until then, only CancelSend can touch them.
func (k msgServer) DelayedSend(ctx context.Context, msg *types.MsgDelayedSend) (*types.MsgDelayedSendResponse, error) {
	sender, err := k.addressCodec.StringToBytes(msg.Creator)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid sender address")
	}
	recipient, err := k.addressCodec.StringToBytes(msg.Recipient)
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidRecipient, err.Error())
	}
	if msg.Creator == msg.Recipient {
		return nil, types.ErrSendToSelf
	}
	if err := msg.Amount.Validate(); err != nil || msg.Amount.IsZero() {
		return nil, types.ErrInvalidAmount
	}
	coins := sdk.NewCoins(msg.Amount)

	// The shop switch: final-only recipients get paid now, or not at all.
	target, err := k.getRecord(ctx, msg.Recipient)
	if err != nil {
		return nil, err
	}
	if target.Finalonly {
		if msg.Window != 0 {
			return nil, types.ErrFinalOnly
		}
		if err := k.bankKeeper.SendCoins(ctx, sender, recipient, coins); err != nil {
			return nil, err
		}
		emit(ctx, "final_send", "sender", msg.Creator, "recipient", msg.Recipient, "amount", coins.String())
		return &types.MsgDelayedSendResponse{Final: true}, nil
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	window := msg.Window
	if window == 0 {
		window = params.Defaultwindow
	}
	if window < params.Minwindow {
		return nil, errorsmod.Wrapf(types.ErrWindowTooShort, "minimum is %d seconds", params.Minwindow)
	}
	if window > params.Maxwindow {
		return nil, errorsmod.Wrapf(types.ErrWindowTooLong, "maximum is %d seconds", params.Maxwindow)
	}

	open, err := k.countOpen(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}
	if open >= types.MaxOpenPerSender {
		return nil, errorsmod.Wrapf(types.ErrTooManyPending, "limit is %d", types.MaxOpenPerSender)
	}

	// Park the coins. This is also where "not enough TFL" is caught.
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, coins); err != nil {
		return nil, err
	}

	id, err := k.PendingSeq.Next(ctx)
	if err != nil {
		return nil, err
	}
	p := types.Pending{
		Id:        id,
		Sender:    msg.Creator,
		Recipient: msg.Recipient,
		Amount:    msg.Amount,
		Executeat: blockTime(ctx) + int64(window),
	}
	if err := k.Pending.Set(ctx, id, p); err != nil {
		return nil, err
	}
	if err := k.bump(ctx, msg.Creator, func(r *types.Record) { r.Sent++ }); err != nil {
		return nil, err
	}

	emit(ctx, "delayed_send",
		"id", fmt.Sprint(id),
		"sender", msg.Creator,
		"recipient", msg.Recipient,
		"amount", coins.String(),
		"execute_at", fmt.Sprint(p.Executeat),
	)
	return &types.MsgDelayedSendResponse{Id: id}, nil
}
