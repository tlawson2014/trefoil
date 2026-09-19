package keeper_test

// The design's "definition of done" for undo, as code:
//
//   - a send waits in holding for its window, then pays out by itself
//   - the sender can cancel during the window; nobody else can
//   - after payout there is nothing to cancel
//   - windows shorter than the chain minimum or longer than the maximum
//     are refused; 0 means the default
//   - a final-only address is paid instantly and can't be sent an undoable
//     payment
//   - the public tally counts sends and cancels
//   - one address can't flood the network with pending sends

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"trefoil/x/undo/keeper"
	"trefoil/x/undo/types"
)

// addr makes a deterministic bech32 address from a single byte.
func (f *fixture) addr(t *testing.T, b byte) (string, sdk.AccAddress) {
	t.Helper()
	raw := make([]byte, 20)
	raw[0] = b
	s, err := f.addressCodec.BytesToString(raw)
	require.NoError(t, err)
	return s, raw
}

func (f *fixture) fund(a sdk.AccAddress, amount int64) {
	f.bank.balances[key(a)] = sdk.NewCoins(sdk.NewInt64Coin("utfl", amount))
}

func (f *fixture) balance(a sdk.AccAddress) int64 {
	return f.bank.balances[key(a)].AmountOf("utfl").Int64()
}

// advance moves block time forward by n seconds and runs the end-blocker,
// like n seconds of blocks passing on the real chain.
func (f *fixture) advance(t *testing.T, n int64) {
	t.Helper()
	f.ctx = f.ctx.WithBlockTime(f.ctx.BlockTime().Add(time.Duration(n) * time.Second))
	require.NoError(t, f.keeper.EndBlocker(f.ctx))
}

func coin(n int64) sdk.Coin { return sdk.NewInt64Coin("utfl", n) }

func TestSendWaitsThenPaysOut(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	alice, aliceAcc := f.addr(t, 1)
	bob, bobAcc := f.addr(t, 2)
	f.fund(aliceAcc, 1000)

	res, err := ms.DelayedSend(f.ctx, &types.MsgDelayedSend{Creator: alice, Recipient: bob, Amount: coin(100), Window: 0})
	require.NoError(t, err)
	require.False(t, res.Final)

	// Coins left alice and sit in holding; bob has nothing yet.
	require.EqualValues(t, 900, f.balance(aliceAcc))
	require.EqualValues(t, 0, f.balance(bobAcc))
	require.EqualValues(t, 100, f.bank.holding.AmountOf("utfl").Int64())

	// Default window is 600 s. One second early: still pending.
	f.advance(t, 599)
	require.EqualValues(t, 0, f.balance(bobAcc))
	_, err = f.keeper.Pending.Get(f.ctx, res.Id)
	require.NoError(t, err)

	// On the dot: paid out and gone.
	f.advance(t, 1)
	require.EqualValues(t, 100, f.balance(bobAcc))
	require.True(t, f.bank.holding.IsZero())
	_, err = f.keeper.Pending.Get(f.ctx, res.Id)
	require.Error(t, err)

	rec, err := f.keeper.Record.Get(f.ctx, alice)
	require.NoError(t, err)
	require.EqualValues(t, 1, rec.Sent)
	require.EqualValues(t, 0, rec.Cancelled)
}

func TestSenderCanUndoDuringWindow(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	alice, aliceAcc := f.addr(t, 1)
	bob, bobAcc := f.addr(t, 2)
	mallory, _ := f.addr(t, 3)
	f.fund(aliceAcc, 1000)

	res, err := ms.DelayedSend(f.ctx, &types.MsgDelayedSend{Creator: alice, Recipient: bob, Amount: coin(100), Window: 300})
	require.NoError(t, err)

	// Someone else can't cancel it — not even the recipient.
	_, err = ms.CancelSend(f.ctx, &types.MsgCancelSend{Creator: mallory, Id: res.Id})
	require.ErrorIs(t, err, types.ErrNotSender)
	_, err = ms.CancelSend(f.ctx, &types.MsgCancelSend{Creator: bob, Id: res.Id})
	require.ErrorIs(t, err, types.ErrNotSender)

	f.advance(t, 299)
	_, err = ms.CancelSend(f.ctx, &types.MsgCancelSend{Creator: alice, Id: res.Id})
	require.NoError(t, err)

	require.EqualValues(t, 1000, f.balance(aliceAcc))
	require.EqualValues(t, 0, f.balance(bobAcc))
	require.True(t, f.bank.holding.IsZero())

	// The tally remembers.
	rec, err := f.keeper.Record.Get(f.ctx, alice)
	require.NoError(t, err)
	require.EqualValues(t, 1, rec.Sent)
	require.EqualValues(t, 1, rec.Cancelled)

	// Time passing does nothing further.
	f.advance(t, 10)
	require.EqualValues(t, 0, f.balance(bobAcc))
}

func TestNothingToUndoAfterPayout(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	alice, aliceAcc := f.addr(t, 1)
	bob, bobAcc := f.addr(t, 2)
	f.fund(aliceAcc, 1000)

	res, err := ms.DelayedSend(f.ctx, &types.MsgDelayedSend{Creator: alice, Recipient: bob, Amount: coin(100), Window: 120})
	require.NoError(t, err)
	f.advance(t, 120)
	require.EqualValues(t, 100, f.balance(bobAcc))

	_, err = ms.CancelSend(f.ctx, &types.MsgCancelSend{Creator: alice, Id: res.Id})
	require.ErrorIs(t, err, types.ErrNoSuchPending)
	require.EqualValues(t, 100, f.balance(bobAcc)) // final means final
}

func TestWindowBounds(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	alice, aliceAcc := f.addr(t, 1)
	bob, _ := f.addr(t, 2)
	f.fund(aliceAcc, 1000)

	_, err := ms.DelayedSend(f.ctx, &types.MsgDelayedSend{Creator: alice, Recipient: bob, Amount: coin(1), Window: 119})
	require.ErrorIs(t, err, types.ErrWindowTooShort)

	_, err = ms.DelayedSend(f.ctx, &types.MsgDelayedSend{Creator: alice, Recipient: bob, Amount: coin(1), Window: 86401})
	require.ErrorIs(t, err, types.ErrWindowTooLong)

	_, err = ms.DelayedSend(f.ctx, &types.MsgDelayedSend{Creator: alice, Recipient: bob, Amount: coin(1), Window: 120})
	require.NoError(t, err)
	_, err = ms.DelayedSend(f.ctx, &types.MsgDelayedSend{Creator: alice, Recipient: bob, Amount: coin(1), Window: 86400})
	require.NoError(t, err)

	// Nothing was taken for the refused ones.
	require.EqualValues(t, 998, f.balance(aliceAcc))
}

func TestBasicChecks(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	alice, aliceAcc := f.addr(t, 1)
	bob, _ := f.addr(t, 2)
	f.fund(aliceAcc, 50)

	_, err := ms.DelayedSend(f.ctx, &types.MsgDelayedSend{Creator: alice, Recipient: alice, Amount: coin(1)})
	require.ErrorIs(t, err, types.ErrSendToSelf)

	_, err = ms.DelayedSend(f.ctx, &types.MsgDelayedSend{Creator: alice, Recipient: "nonsense", Amount: coin(1)})
	require.ErrorIs(t, err, types.ErrInvalidRecipient)

	_, err = ms.DelayedSend(f.ctx, &types.MsgDelayedSend{Creator: alice, Recipient: bob, Amount: coin(0)})
	require.ErrorIs(t, err, types.ErrInvalidAmount)

	_, err = ms.DelayedSend(f.ctx, &types.MsgDelayedSend{Creator: alice, Recipient: bob, Amount: coin(51)})
	require.ErrorContains(t, err, "insufficient funds")
	require.EqualValues(t, 50, f.balance(aliceAcc))
}

func TestFinalOnlyRecipient(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	alice, aliceAcc := f.addr(t, 1)
	shop, shopAcc := f.addr(t, 2)
	f.fund(aliceAcc, 1000)

	_, err := ms.SetFinalOnly(f.ctx, &types.MsgSetFinalOnly{Creator: shop, Finalonly: true})
	require.NoError(t, err)

	// An undoable payment to the shop is refused outright.
	_, err = ms.DelayedSend(f.ctx, &types.MsgDelayedSend{Creator: alice, Recipient: shop, Amount: coin(100), Window: 600})
	require.ErrorIs(t, err, types.ErrFinalOnly)
	require.EqualValues(t, 1000, f.balance(aliceAcc))

	// Window 0 is the sender saying "I know, pay now".
	res, err := ms.DelayedSend(f.ctx, &types.MsgDelayedSend{Creator: alice, Recipient: shop, Amount: coin(100), Window: 0})
	require.NoError(t, err)
	require.True(t, res.Final)
	require.EqualValues(t, 100, f.balance(shopAcc))
	require.True(t, f.bank.holding.IsZero())

	// Nothing pending, so nothing to undo.
	_, err = ms.CancelSend(f.ctx, &types.MsgCancelSend{Creator: alice, Id: res.Id})
	require.ErrorIs(t, err, types.ErrNoSuchPending)

	// Switch it off again and undoable sends work as normal.
	_, err = ms.SetFinalOnly(f.ctx, &types.MsgSetFinalOnly{Creator: shop, Finalonly: false})
	require.NoError(t, err)
	res, err = ms.DelayedSend(f.ctx, &types.MsgDelayedSend{Creator: alice, Recipient: shop, Amount: coin(100), Window: 600})
	require.NoError(t, err)
	require.False(t, res.Final)
}

func TestTooManyPending(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	alice, aliceAcc := f.addr(t, 1)
	bob, _ := f.addr(t, 2)
	f.fund(aliceAcc, 1000)

	for i := 0; i < types.MaxOpenPerSender; i++ {
		_, err := ms.DelayedSend(f.ctx, &types.MsgDelayedSend{Creator: alice, Recipient: bob, Amount: coin(1)})
		require.NoError(t, err)
	}
	_, err := ms.DelayedSend(f.ctx, &types.MsgDelayedSend{Creator: alice, Recipient: bob, Amount: coin(1)})
	require.ErrorIs(t, err, types.ErrTooManyPending)

	// Once they pay out, the slots free up.
	f.advance(t, 600)
	_, err = ms.DelayedSend(f.ctx, &types.MsgDelayedSend{Creator: alice, Recipient: bob, Amount: coin(1)})
	require.NoError(t, err)
}
