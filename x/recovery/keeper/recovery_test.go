package keeper_test

import (
	"context"
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"trefoil/x/recovery/keeper"
	module "trefoil/x/recovery/module"
	"trefoil/x/recovery/types"
)

// These tests walk the exact "definition of done" list from the design doc.
// They run the module's logic directly, with no network and no real bank —
// a fake in-memory bank stands in — so they finish in milliseconds.

// ---------------------------------------------------------------------------
// Test harness
// ---------------------------------------------------------------------------

// fakeBank is the smallest possible stand-in for the real bank module.
type fakeBank struct {
	balances map[string]sdk.Coins
}

func newFakeBank() *fakeBank { return &fakeBank{balances: map[string]sdk.Coins{}} }

func (b *fakeBank) SpendableCoins(_ context.Context, a sdk.AccAddress) sdk.Coins {
	return b.balances[a.String()]
}
func (b *fakeBank) GetAllBalances(_ context.Context, a sdk.AccAddress) sdk.Coins {
	return b.balances[a.String()]
}
func (b *fakeBank) SendCoins(_ context.Context, from, to sdk.AccAddress, amt sdk.Coins) error {
	b.balances[from.String()] = b.balances[from.String()].Sub(amt...)
	b.balances[to.String()] = b.balances[to.String()].Add(amt...)
	return nil
}

type harness struct {
	t      *testing.T
	ctx    sdk.Context
	k      keeper.Keeper
	msg    types.MsgServer
	bank   *fakeBank
	params types.Params
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx
	authority := authtypes.NewModuleAddress(types.GovModuleName)

	bank := newFakeBank()
	k := keeper.NewKeeper(storeService, encCfg.Codec, addressCodec, authority, nil, bank)

	// Short timings so the tests can "wait" by simply moving block time.
	params := types.NewParams(60 /* delay */, 3600 /* expiry */)
	require.NoError(t, k.Params.Set(ctx, params))

	return &harness{
		t:      t,
		ctx:    ctx.WithBlockTime(time.Unix(1_000_000, 0)),
		k:      k,
		msg:    keeper.NewMsgServerImpl(k),
		bank:   bank,
		params: params,
	}
}

// advance moves block time forward by n seconds, like waiting for blocks.
func (h *harness) advance(n int64) {
	h.ctx = h.ctx.WithBlockTime(h.ctx.BlockTime().Add(time.Duration(n) * time.Second))
}

// addr makes a deterministic, valid bech32 address from a small number.
func addr(i byte) string {
	b := make([]byte, 20)
	b[0] = i
	return sdk.AccAddress(b).String()
}

// Cast: the people in our story.
var (
	tom    = addr(1) // the account that will lose its key
	newTom = addr(2) // Tom's fresh wallet after losing the key
	g1     = addr(11)
	g2     = addr(12)
	g3     = addr(13)
	rando  = addr(99) // not a guardian of anyone
	threeG = []string{g1, g2, g3}
)

func (h *harness) setGuardians(owner string, guardians []string, threshold uint64) error {
	_, err := h.msg.SetGuardians(h.ctx, &types.MsgSetGuardians{Creator: owner, Guardians: guardians, Threshold: threshold})
	return err
}
func (h *harness) request(by, account, newAddr string) error {
	_, err := h.msg.RequestRecovery(h.ctx, &types.MsgRequestRecovery{Creator: by, Account: account, Newaddress: newAddr})
	return err
}
func (h *harness) approve(by, account string) error {
	_, err := h.msg.ApproveRecovery(h.ctx, &types.MsgApproveRecovery{Creator: by, Account: account})
	return err
}
func (h *harness) cancel(by, account string) error {
	_, err := h.msg.CancelRecovery(h.ctx, &types.MsgCancelRecovery{Creator: by, Account: account})
	return err
}
func (h *harness) endBlock() { require.NoError(h.t, h.k.EndBlocker(h.ctx)) }

func (h *harness) fund(who string, amount int64) {
	a, _ := sdk.AccAddressFromBech32(who)
	h.bank.balances[a.String()] = sdk.NewCoins(sdk.NewInt64Coin("utfl", amount))
}
func (h *harness) balance(who string) int64 {
	a, _ := sdk.AccAddressFromBech32(who)
	return h.bank.balances[a.String()].AmountOf("utfl").Int64()
}

// ---------------------------------------------------------------------------
// Setting guardians
// ---------------------------------------------------------------------------

func TestSetGuardians_HappyPath(t *testing.T) {
	h := newHarness(t)
	require.NoError(t, h.setGuardians(tom, threeG, 2))

	gs, err := h.k.Guardianset.Get(h.ctx, tom)
	require.NoError(t, err)
	require.Equal(t, tom, gs.Owner)
	require.Equal(t, uint64(2), gs.Threshold)
	require.Equal(t, threeG, gs.Guardians)
}

func TestSetGuardians_Rules(t *testing.T) {
	h := newHarness(t)

	require.ErrorIs(t, h.setGuardians(tom, nil, 1), types.ErrNoGuardians)
	require.ErrorIs(t, h.setGuardians(tom, []string{g1, g1, g2}, 2), types.ErrDuplicateGuardian)
	require.ErrorIs(t, h.setGuardians(tom, []string{g1, tom}, 2), types.ErrSelfGuardian)
	require.Error(t, h.setGuardians(tom, []string{"not-an-address"}, 1))

	eight := make([]string, 8)
	for i := range eight {
		eight[i] = addr(byte(20 + i))
	}
	require.ErrorIs(t, h.setGuardians(tom, eight, 5), types.ErrTooManyGuardians)

	// Threshold must be a strict majority and at most n.
	require.ErrorIs(t, h.setGuardians(tom, threeG, 0), types.ErrBadThreshold)
	require.ErrorIs(t, h.setGuardians(tom, threeG, 1), types.ErrBadThreshold) // 1 of 3 is not a majority
	require.ErrorIs(t, h.setGuardians(tom, threeG, 4), types.ErrBadThreshold)
	require.NoError(t, h.setGuardians(tom, threeG, 3))
	require.NoError(t, h.setGuardians(tom, threeG, 2)) // replacing is allowed when nothing is pending
}

func TestSetGuardians_BlockedWhileRecoveryPending(t *testing.T) {
	h := newHarness(t)
	require.NoError(t, h.setGuardians(tom, threeG, 2))
	require.NoError(t, h.request(g1, tom, newTom))

	// A thief holding Tom's key cannot quietly swap the guardians mid-recovery.
	require.ErrorIs(t, h.setGuardians(tom, []string{rando}, 1), types.ErrRecoveryPending)
}

// ---------------------------------------------------------------------------
// The full recovery — the design doc's definition of done
// ---------------------------------------------------------------------------

func TestRecovery_EndToEnd(t *testing.T) {
	h := newHarness(t)
	h.fund(tom, 5_000_000) // 5 TFL
	require.NoError(t, h.setGuardians(tom, threeG, 2))

	// Tom loses his key. He makes newTom and tells guardian 1.
	require.NoError(t, h.request(g1, tom, newTom))
	rec, err := h.k.Recovery.Get(h.ctx, tom)
	require.NoError(t, err)
	require.Equal(t, []string{g1}, rec.Approvals)
	require.Zero(t, rec.Executeat, "one approval of two: waiting period must not have started")

	// A second guardian approves -> threshold met -> 60s clock starts.
	require.NoError(t, h.approve(g2, tom))
	rec, _ = h.k.Recovery.Get(h.ctx, tom)
	require.Equal(t, h.ctx.BlockTime().Unix()+60, rec.Executeat)

	// Too early: nothing moves.
	h.advance(30)
	h.endBlock()
	require.Equal(t, int64(5_000_000), h.balance(tom))
	require.Equal(t, int64(0), h.balance(newTom))

	// Delay over: balance moves, guardians follow, record closes.
	h.advance(31)
	h.endBlock()
	require.Equal(t, int64(0), h.balance(tom))
	require.Equal(t, int64(5_000_000), h.balance(newTom))

	_, err = h.k.Recovery.Get(h.ctx, tom)
	require.Error(t, err, "recovery record should be gone")

	_, err = h.k.Guardianset.Get(h.ctx, tom)
	require.Error(t, err, "old account's guardian set should be gone")

	gs, err := h.k.Guardianset.Get(h.ctx, newTom)
	require.NoError(t, err, "new account inherits the guardian set")
	require.Equal(t, newTom, gs.Owner)
	require.Equal(t, threeG, gs.Guardians)
}

func TestRecovery_OwnerCanCancelDuringDelay(t *testing.T) {
	h := newHarness(t)
	h.fund(tom, 1_000_000)
	require.NoError(t, h.setGuardians(tom, threeG, 2))
	require.NoError(t, h.request(g1, tom, newTom))
	require.NoError(t, h.approve(g2, tom))

	// Tom's key wasn't lost after all (or two guardians were tricked).
	// Tom vetoes with his own key inside the 60s window.
	h.advance(10)
	require.NoError(t, h.cancel(tom, tom))

	h.advance(100)
	h.endBlock()
	require.Equal(t, int64(1_000_000), h.balance(tom), "nothing should have moved")
	require.Equal(t, int64(0), h.balance(newTom))
}

func TestRecovery_OnlyOwnerCanCancel(t *testing.T) {
	h := newHarness(t)
	require.NoError(t, h.setGuardians(tom, threeG, 2))
	require.NoError(t, h.request(g1, tom, newTom))

	require.ErrorIs(t, h.cancel(g1, tom), types.ErrNotAccountOwner)
	require.ErrorIs(t, h.cancel(rando, tom), types.ErrNotAccountOwner)
	require.ErrorIs(t, h.cancel(tom, g1), types.ErrNotAccountOwner) // creator != account
}

func TestRecovery_OneGuardianIsNotEnough(t *testing.T) {
	h := newHarness(t)
	h.fund(tom, 1_000_000)
	require.NoError(t, h.setGuardians(tom, threeG, 2))
	require.NoError(t, h.request(g1, tom, newTom))

	// Only one approval; even after a long time nothing executes.
	h.advance(600)
	h.endBlock()
	require.Equal(t, int64(1_000_000), h.balance(tom))

	// ...and after the expiry the request is simply dropped.
	h.advance(3600)
	h.endBlock()
	_, err := h.k.Recovery.Get(h.ctx, tom)
	require.Error(t, err, "expired request should be removed")
}

// ---------------------------------------------------------------------------
// Guardian rules around requests and approvals
// ---------------------------------------------------------------------------

func TestRequest_Rules(t *testing.T) {
	h := newHarness(t)

	// No guardian set yet -> cannot be recovered.
	require.ErrorIs(t, h.request(g1, tom, newTom), types.ErrNoGuardianSet)

	require.NoError(t, h.setGuardians(tom, threeG, 2))
	require.ErrorIs(t, h.request(rando, tom, newTom), types.ErrNotGuardian)
	require.ErrorIs(t, h.request(g1, tom, tom), types.ErrBadNewAddress)
	require.ErrorIs(t, h.request(g1, tom, "garbage"), types.ErrBadNewAddress)

	require.NoError(t, h.request(g1, tom, newTom))
	require.ErrorIs(t, h.request(g2, tom, newTom), types.ErrRecoveryPending)
}

func TestApprove_Rules(t *testing.T) {
	h := newHarness(t)
	require.NoError(t, h.setGuardians(tom, threeG, 2))

	require.ErrorIs(t, h.approve(g2, tom), types.ErrNoRecoveryPending)

	require.NoError(t, h.request(g1, tom, newTom))
	require.ErrorIs(t, h.approve(g1, tom), types.ErrAlreadyApproved) // requester already counted
	require.ErrorIs(t, h.approve(rando, tom), types.ErrNotGuardian)

	require.NoError(t, h.approve(g2, tom))
	require.ErrorIs(t, h.approve(g3, tom), types.ErrRecoveryLocked) // threshold met; locked
}

func TestApprove_ExpiredRequestIsRejected(t *testing.T) {
	h := newHarness(t)
	require.NoError(t, h.setGuardians(tom, threeG, 2))
	require.NoError(t, h.request(g1, tom, newTom))

	h.advance(3601) // past Requestexpiry
	require.ErrorIs(t, h.approve(g2, tom), types.ErrRecoveryExpired)

	// A fresh request can now be opened.
	require.NoError(t, h.request(g1, tom, newTom))
}

func TestSingleGuardian_RequestIsApproval(t *testing.T) {
	h := newHarness(t)
	h.fund(tom, 42)
	require.NoError(t, h.setGuardians(tom, []string{g1}, 1))
	require.NoError(t, h.request(g1, tom, newTom))

	rec, _ := h.k.Recovery.Get(h.ctx, tom)
	require.NotZero(t, rec.Executeat, "threshold 1: the request itself starts the clock")

	h.advance(61)
	h.endBlock()
	require.Equal(t, int64(42), h.balance(newTom))
}
