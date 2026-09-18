// Package ante holds Trefoil's one custom transaction check: letting a
// brand-new wallet choose its guardians before it has ever received a coin.
package ante

import (
	"bytes"
	"context"

	errorsmod "cosmossdk.io/errors"
	txsigning "cosmossdk.io/x/tx/signing"
	"google.golang.org/protobuf/types/known/anypb"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"

	"trefoil/x/recovery/types"
)

// The problem this solves
//
// On a stock Cosmos chain an address doesn't exist until it receives coins,
// and an address that doesn't exist can't sign anything. So a fresh wallet
// couldn't set guardians until someone sent it money — a hole in "protected
// from the moment you sign up".
//
// The fix: a transaction that contains exactly one MsgSetGuardians, from an
// address the chain has never seen, is allowed to CREATE that account. The
// wallet signs it with account number 0 and sequence 0 (the only values it
// can know in advance); we verify the signature on that basis, create the
// account with the sender's public key, and let the message run.
//
// Every other transaction goes through the standard checks untouched.
//
// Spam guard: creating accounts costs the network storage, and this path is
// free. So the recovery keeper caps how many accounts may be bootstrapped
// per block. Past the cap, the transaction is rejected and the wallet retries
// next block.

// MaxBootstrapPerBlock is how many never-seen addresses may create
// themselves via set-guardians in one block.
const MaxBootstrapPerBlock = 100

// AccountKeeper is the slice of x/auth this decorator needs.
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	SetAccount(ctx context.Context, acc sdk.AccountI)
}

// BootstrapCounter is the per-block spam guard, implemented by the recovery keeper.
type BootstrapCounter interface {
	// CountBootstrap records one more bootstrapped account in this block, or
	// returns an error if the block is already at the cap.
	CountBootstrap(ctx sdk.Context, max uint64) error
}

// GuardianBootstrapDecorator lets a never-seen address submit MsgSetGuardians.
type GuardianBootstrapDecorator struct {
	ak      AccountKeeper
	counter BootstrapCounter
	handler *txsigning.HandlerMap
}

func NewGuardianBootstrapDecorator(ak AccountKeeper, counter BootstrapCounter, handler *txsigning.HandlerMap) GuardianBootstrapDecorator {
	return GuardianBootstrapDecorator{ak: ak, counter: counter, handler: handler}
}

// AnteHandle either hands the transaction on unchanged (the normal case) or,
// for a bootstrap transaction, does the whole check itself and does NOT call
// next — the standard fee and signature decorators would reject an account
// that doesn't exist yet, and this path has already done their job.
func (d GuardianBootstrapDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	creator, ok := bootstrapCandidate(tx)
	if !ok {
		return next(ctx, tx, simulate)
	}
	if d.ak.GetAccount(ctx, creator) != nil {
		// Account exists: this is just an ordinary set-guardians. Normal path.
		return next(ctx, tx, simulate)
	}

	// --- Bootstrap path: a never-seen address is choosing its guardians. ---

	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return ctx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "invalid transaction type")
	}
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}
	if !feeTx.GetFee().IsZero() {
		// A fee would need an account to pay it from. Trefoil is feeless anyway.
		return ctx, errorsmod.Wrap(types.ErrBootstrapRejected, "a first set-guardians transaction must carry no fee")
	}

	signers, err := sigTx.GetSigners()
	if err != nil {
		return ctx, err
	}
	pubkeys, err := sigTx.GetPubKeys()
	if err != nil {
		return ctx, err
	}
	sigs, err := sigTx.GetSignaturesV2()
	if err != nil {
		return ctx, err
	}
	if len(signers) != 1 || len(pubkeys) != 1 || len(sigs) != 1 {
		return ctx, errorsmod.Wrap(types.ErrBootstrapRejected, "exactly one signer required")
	}
	if !bytes.Equal(signers[0], creator) {
		return ctx, errorsmod.Wrap(types.ErrBootstrapRejected, "signer must be the account setting guardians")
	}
	pk := pubkeys[0]
	if pk == nil {
		return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidPubKey, "public key required for a first transaction")
	}
	if !bytes.Equal(pk.Address(), creator) {
		return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidPubKey, "public key does not match the account")
	}
	if sigs[0].Sequence != 0 {
		return ctx, errorsmod.Wrap(sdkerrors.ErrWrongSequence, "a first transaction must use sequence 0")
	}

	// Spam guard, only when the block is really being built.
	if !simulate {
		if err := d.counter.CountBootstrap(ctx, MaxBootstrapPerBlock); err != nil {
			return ctx, err
		}
	}

	// Verify the signature exactly as the standard decorator would, but with
	// account number 0 — the number the wallet was forced to guess.
	if !simulate && !ctx.IsReCheckTx() && ctx.IsSigverifyTx() {
		anyPk, err := codectypes.NewAnyWithValue(pk)
		if err != nil {
			return ctx, err
		}
		signerData := txsigning.SignerData{
			Address:       creator.String(),
			ChainID:       ctx.ChainID(),
			AccountNumber: 0,
			Sequence:      0,
			PubKey:        &anypb.Any{TypeUrl: anyPk.TypeUrl, Value: anyPk.Value},
		}
		adaptable, ok := tx.(authsigning.V2AdaptableTx)
		if !ok {
			return ctx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "tx does not support signing v2")
		}
		if err := authsigning.VerifySignature(ctx, pk, signerData, sigs[0].Data, d.handler, adaptable.GetSigningTxData()); err != nil {
			return ctx, errorsmod.Wrap(sdkerrors.ErrUnauthorized,
				"signature verification failed; a first transaction must be signed with account number 0, sequence 0, and the right chain-id")
		}
	}

	// Create the account, attach the key, and mark this first transaction as
	// used so it can never be replayed.
	acc := d.ak.NewAccountWithAddress(ctx, creator)
	if err := acc.SetPubKey(pk); err != nil {
		return ctx, err
	}
	if err := acc.SetSequence(1); err != nil {
		return ctx, err
	}
	d.ak.SetAccount(ctx, acc)

	ctx.EventManager().EmitEvent(sdk.NewEvent("account_bootstrapped",
		sdk.NewAttribute("address", creator.String()),
	))

	// Deliberately not calling next: the rest of the chain is fee and
	// signature checks for accounts that already exist.
	return ctx, nil
}

// bootstrapCandidate reports whether tx is exactly one MsgSetGuardians, and
// if so, the address setting them.
func bootstrapCandidate(tx sdk.Tx) (sdk.AccAddress, bool) {
	msgs := tx.GetMsgs()
	if len(msgs) != 1 {
		return nil, false
	}
	msg, ok := msgs[0].(*types.MsgSetGuardians)
	if !ok {
		return nil, false
	}
	addr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, false
	}
	return addr, true
}
