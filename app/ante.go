package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"

	recoveryante "trefoil/x/recovery/ante"
)

// newAnteHandler builds Trefoil's transaction-check chain.
//
// It is the Cosmos SDK's standard chain (see x/auth/ante.NewAnteHandler)
// with ONE addition: the guardian-bootstrap decorator, placed just before
// the fee check. That decorator lets a never-seen address submit its first
// set-guardians transaction with zero coins; for every other transaction it
// does nothing and the standard checks run unchanged.
//
// Order matters. The bootstrap decorator sits after the cheap, stateless
// checks (basic validity, memo, tx size) and before anything that assumes
// the signer's account already exists (fee deduction, signature checks).
func newAnteHandler(app *App) (sdk.AnteHandler, error) {
	opts := ante.HandlerOptions{
		AccountKeeper:   app.AuthKeeper,
		BankKeeper:      app.BankKeeper,
		SignModeHandler: app.txConfig.SignModeHandler(),
		SigGasConsumer:  ante.DefaultSigVerificationGasConsumer,
	}

	decorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(),
		ante.NewExtensionOptionsDecorator(opts.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(opts.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(opts.AccountKeeper),

		// Trefoil: a new wallet's first set-guardians creates its account.
		recoveryante.NewGuardianBootstrapDecorator(app.AuthKeeper, app.RecoveryKeeper, opts.SignModeHandler),

		ante.NewDeductFeeDecorator(opts.AccountKeeper, opts.BankKeeper, opts.FeegrantKeeper, opts.TxFeeChecker),
		ante.NewSetPubKeyDecorator(opts.AccountKeeper),
		ante.NewValidateSigCountDecorator(opts.AccountKeeper),
		ante.NewSigGasConsumeDecorator(opts.AccountKeeper, opts.SigGasConsumer),
		ante.NewSigVerificationDecorator(opts.AccountKeeper, opts.SignModeHandler),
		ante.NewIncrementSequenceDecorator(opts.AccountKeeper),
	}

	return sdk.ChainAnteDecorators(decorators...), nil
}
