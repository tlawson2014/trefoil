package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// Every reason an undo transaction can be refused. The codes are stable so
// wallets can match on them; the text is what a person sees.
var (
	ErrInvalidSigner    = errors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidRecipient = errors.Register(ModuleName, 1101, "recipient is not a valid address")
	ErrSendToSelf       = errors.Register(ModuleName, 1102, "cannot send to yourself")
	ErrInvalidAmount    = errors.Register(ModuleName, 1103, "amount must be a positive number of coins")
	ErrWindowTooShort   = errors.Register(ModuleName, 1104, "undo window is shorter than the chain minimum")
	ErrWindowTooLong    = errors.Register(ModuleName, 1105, "undo window is longer than the chain maximum")
	ErrFinalOnly        = errors.Register(ModuleName, 1106, "this address only accepts final payments: resend with window 0")
	ErrTooManyPending   = errors.Register(ModuleName, 1107, "too many undoable sends in flight from this address")
	ErrNoSuchPending    = errors.Register(ModuleName, 1108, "no pending send with that id")
	ErrNotSender        = errors.Register(ModuleName, 1109, "only the sender can undo a send")
)
