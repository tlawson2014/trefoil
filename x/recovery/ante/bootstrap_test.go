package ante_test

import (
	"context"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	recoveryante "trefoil/x/recovery/ante"
	module "trefoil/x/recovery/module"
	"trefoil/x/recovery/types"
)

// These tests prove the one thing the bootstrap path is for — a wallet with
// no coins can set guardians as its very first act — and that every way of
// abusing that path is refused.

// ---------------------------------------------------------------------------
// Fakes: the smallest account keeper and counter that behave like the real ones
// ---------------------------------------------------------------------------

type fakeAccounts struct {
	accs   map[string]sdk.AccountI
	nextNo uint64
}

func newFakeAccounts() *fakeAccounts { return &fakeAccounts{accs: map[string]sdk.AccountI{}} }

func (f *fakeAccounts) GetAccount(_ context.Context, addr sdk.AccAddress) sdk.AccountI {
	return f.accs[addr.String()]
}
func (f *fakeAccounts) NewAccountWithAddress(_ context.Context, addr sdk.AccAddress) sdk.AccountI {
	f.nextNo++
	// Real account numbers are never 0 after genesis — that's the whole
	// reason the wallet can't guess them.
	return authtypes.NewBaseAccount(addr, nil, 40+f.nextNo, 0)
}
func (f *fakeAccounts) SetAccount(_ context.Context, acc sdk.AccountI) {
	f.accs[acc.GetAddress().String()] = acc
}

type fakeCounter struct {
	n    uint64
	fail bool
}

func (c *fakeCounter) CountBootstrap(_ sdk.Context, max uint64) error {
	if c.fail || c.n >= max {
		return types.ErrTooManyNewAccounts
	}
	c.n++
	return nil
}

// ---------------------------------------------------------------------------
// Harness
// ---------------------------------------------------------------------------

type harness struct {
	t        *testing.T
	ctx      sdk.Context
	enc      moduletestutil.TestEncodingConfig
	accounts *fakeAccounts
	counter  *fakeCounter
	handler  sdk.AnteHandler
	nextHit  bool // did the decorator hand off to the rest of the chain?
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	enc := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	banktypes.RegisterInterfaces(enc.InterfaceRegistry)

	key := storetypes.NewKVStoreKey("test")
	ctx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("t")).Ctx.
		WithChainID("trefoil-test").WithBlockHeight(10)

	h := &harness{t: t, ctx: ctx, enc: enc, accounts: newFakeAccounts(), counter: &fakeCounter{}}
	dec := recoveryante.NewGuardianBootstrapDecorator(h.accounts, h.counter, enc.TxConfig.SignModeHandler())
	next := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		h.nextHit = true
		return ctx, nil
	}
	h.handler = func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		h.nextHit = false
		return dec.AnteHandle(ctx, tx, simulate, next)
	}
	return h
}

type signer struct {
	priv cryptotypes.PrivKey
	addr sdk.AccAddress
}

func newSigner() signer {
	priv := secp256k1.GenPrivKey()
	return signer{priv: priv, addr: sdk.AccAddress(priv.PubKey().Address())}
}

// signedTx builds and signs a tx the way a wallet would: with whatever
// account number and sequence the caller claims.
func (h *harness) signedTx(s signer, accNum, seq uint64, fee sdk.Coins, msgs ...sdk.Msg) sdk.Tx {
	h.t.Helper()
	b := h.enc.TxConfig.NewTxBuilder()
	require.NoError(h.t, b.SetMsgs(msgs...))
	b.SetFeeAmount(fee)
	b.SetGasLimit(200000)

	mode := signing.SignMode_SIGN_MODE_DIRECT
	// Round 1: placeholder signature so the sign bytes can be computed.
	require.NoError(h.t, b.SetSignatures(signing.SignatureV2{
		PubKey:   s.priv.PubKey(),
		Data:     &signing.SingleSignatureData{SignMode: mode},
		Sequence: seq,
	}))
	signerData := authsigning.SignerData{
		ChainID:       h.ctx.ChainID(),
		AccountNumber: accNum,
		Sequence:      seq,
		PubKey:        s.priv.PubKey(),
		Address:       s.addr.String(),
	}
	bytesToSign, err := authsigning.GetSignBytesAdapter(context.Background(), h.enc.TxConfig.SignModeHandler(), mode, signerData, b.GetTx())
	require.NoError(h.t, err)
	sig, err := s.priv.Sign(bytesToSign)
	require.NoError(h.t, err)
	// Round 2: the real signature.
	require.NoError(h.t, b.SetSignatures(signing.SignatureV2{
		PubKey:   s.priv.PubKey(),
		Data:     &signing.SingleSignatureData{SignMode: mode, Signature: sig},
		Sequence: seq,
	}))
	return b.GetTx()
}

func setGuardiansMsg(s signer) *types.MsgSetGuardians {
	return &types.MsgSetGuardians{
		Creator:   s.addr.String(),
		Guardians: []string{newSigner().addr.String(), newSigner().addr.String(), newSigner().addr.String()},
		Threshold: 2,
	}
}

// ---------------------------------------------------------------------------
// The point: a coinless wallet can set guardians
// ---------------------------------------------------------------------------

func TestBootstrap_NewAccountCanSetGuardians(t *testing.T) {
	h := newHarness(t)
	tom := newSigner()

	// Tom has never touched the chain. He signs with the only numbers he can
	// know: account 0, sequence 0.
	tx := h.signedTx(tom, 0, 0, nil, setGuardiansMsg(tom))
	_, err := h.handler(h.ctx, tx, false)
	require.NoError(t, err)

	require.False(t, h.nextHit, "bootstrap path must not fall through to the standard fee/sig checks")

	acc := h.accounts.GetAccount(h.ctx, tom.addr)
	require.NotNil(t, acc, "account should now exist")
	require.NotNil(t, acc.GetPubKey(), "public key should be attached")
	require.Equal(t, uint64(1), acc.GetSequence(), "first transaction consumed; sequence must be 1 so it can't be replayed")
	require.Equal(t, uint64(1), h.counter.n, "counted against the per-block cap")
}

func TestBootstrap_ReplayIsRejected(t *testing.T) {
	h := newHarness(t)
	tom := newSigner()
	tx := h.signedTx(tom, 0, 0, nil, setGuardiansMsg(tom))

	_, err := h.handler(h.ctx, tx, false)
	require.NoError(t, err)

	// Same signed bytes again: the account now exists, so it goes down the
	// normal path — which (in the real chain) fails on sequence. Here we
	// just prove the bootstrap path doesn't swallow it a second time.
	_, err = h.handler(h.ctx, tx, false)
	require.NoError(t, err)
	require.True(t, h.nextHit, "existing account must take the standard path")
	require.Equal(t, uint64(1), h.counter.n, "no second bootstrap")
}

// ---------------------------------------------------------------------------
// Everything else takes the normal path
// ---------------------------------------------------------------------------

func TestBootstrap_ExistingAccountIsNormal(t *testing.T) {
	h := newHarness(t)
	tom := newSigner()
	h.accounts.SetAccount(h.ctx, authtypes.NewBaseAccount(tom.addr, tom.priv.PubKey(), 7, 3))

	tx := h.signedTx(tom, 7, 3, nil, setGuardiansMsg(tom))
	_, err := h.handler(h.ctx, tx, false)
	require.NoError(t, err)
	require.True(t, h.nextHit)
	require.Equal(t, uint64(0), h.counter.n)
}

func TestBootstrap_OtherMessagesAreNormal(t *testing.T) {
	h := newHarness(t)
	tom := newSigner()

	// A bank send from a never-seen account is NOT bootstrapped: it goes to
	// the standard checks, which will reject it (no account, no coins).
	send := &banktypes.MsgSend{FromAddress: tom.addr.String(), ToAddress: newSigner().addr.String(), Amount: sdk.NewCoins(sdk.NewInt64Coin("utfl", 1))}
	tx := h.signedTx(tom, 0, 0, nil, send)
	_, err := h.handler(h.ctx, tx, false)
	require.NoError(t, err)
	require.True(t, h.nextHit)
	require.Nil(t, h.accounts.GetAccount(h.ctx, tom.addr), "no account created for a non-bootstrap tx")

	// Two messages, even if one is set-guardians: normal path.
	tx = h.signedTx(tom, 0, 0, nil, setGuardiansMsg(tom), send)
	_, err = h.handler(h.ctx, tx, false)
	require.NoError(t, err)
	require.True(t, h.nextHit)
	require.Nil(t, h.accounts.GetAccount(h.ctx, tom.addr))
}

// ---------------------------------------------------------------------------
// Abuse
// ---------------------------------------------------------------------------

func TestBootstrap_WrongKeyIsRejected(t *testing.T) {
	h := newHarness(t)
	tom := newSigner()
	thief := newSigner()

	// The thief signs a set-guardians naming Tom's address as creator.
	// Signer (from the message) is Tom; the key is the thief's.
	tx := h.signedTx(thief, 0, 0, nil, setGuardiansMsg(tom))
	_, err := h.handler(h.ctx, tx, false)
	require.Error(t, err)
	require.Nil(t, h.accounts.GetAccount(h.ctx, tom.addr))
	require.Equal(t, uint64(0), h.counter.n, "rejected before counting")
}

func TestBootstrap_BadSignatureIsRejected(t *testing.T) {
	h := newHarness(t)
	tom := newSigner()

	// Right key, but signed as if for account number 5 — the bytes don't
	// match what the chain verifies against (account 0).
	tx := h.signedTx(tom, 5, 0, nil, setGuardiansMsg(tom))
	_, err := h.handler(h.ctx, tx, false)
	require.ErrorContains(t, err, "signature verification failed")
	require.Nil(t, h.accounts.GetAccount(h.ctx, tom.addr))
}

func TestBootstrap_WrongSequenceIsRejected(t *testing.T) {
	h := newHarness(t)
	tom := newSigner()
	tx := h.signedTx(tom, 0, 1, nil, setGuardiansMsg(tom))
	_, err := h.handler(h.ctx, tx, false)
	require.Error(t, err)
	require.Nil(t, h.accounts.GetAccount(h.ctx, tom.addr))
}

func TestBootstrap_FeeIsRejected(t *testing.T) {
	h := newHarness(t)
	tom := newSigner()
	tx := h.signedTx(tom, 0, 0, sdk.NewCoins(sdk.NewInt64Coin("utfl", 1)), setGuardiansMsg(tom))
	_, err := h.handler(h.ctx, tx, false)
	require.ErrorIs(t, err, types.ErrBootstrapRejected)
}

func TestBootstrap_PerBlockCapHolds(t *testing.T) {
	h := newHarness(t)
	h.counter.n = recoveryante.MaxBootstrapPerBlock // block already full

	tom := newSigner()
	tx := h.signedTx(tom, 0, 0, nil, setGuardiansMsg(tom))
	_, err := h.handler(h.ctx, tx, false)
	require.ErrorIs(t, err, types.ErrTooManyNewAccounts)
	require.Nil(t, h.accounts.GetAccount(h.ctx, tom.addr), "no account when the block is full")
}

func TestBootstrap_SimulateDoesNotCount(t *testing.T) {
	h := newHarness(t)
	tom := newSigner()
	tx := h.signedTx(tom, 0, 0, nil, setGuardiansMsg(tom))
	_, err := h.handler(h.ctx, tx, true)
	require.NoError(t, err)
	require.Equal(t, uint64(0), h.counter.n, "gas estimation must not eat into the per-block cap")
}
