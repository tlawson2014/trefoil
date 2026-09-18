package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/recovery module sentinel errors.
//
// Every way a recovery transaction can be rejected has a named error here.
// The number is a stable code that wallets and explorers can match on; the
// text is what a person sees. Codes 1100+ are the module's own range.
var (
	ErrInvalidSigner = errors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")

	// Guardian set rules
	ErrNoGuardians       = errors.Register(ModuleName, 1101, "at least one guardian is required")
	ErrTooManyGuardians  = errors.Register(ModuleName, 1102, "too many guardians (max 7)")
	ErrDuplicateGuardian = errors.Register(ModuleName, 1103, "the same guardian is listed twice")
	ErrSelfGuardian      = errors.Register(ModuleName, 1104, "an account cannot be its own guardian")
	ErrBadThreshold      = errors.Register(ModuleName, 1105, "threshold must be more than half the guardians and at most all of them")
	ErrNoGuardianSet     = errors.Register(ModuleName, 1106, "this account has no guardians set, so it cannot be recovered")

	// Recovery flow rules
	ErrNotGuardian       = errors.Register(ModuleName, 1110, "signer is not a guardian of this account")
	ErrRecoveryPending   = errors.Register(ModuleName, 1111, "a recovery is already pending for this account")
	ErrNoRecoveryPending = errors.Register(ModuleName, 1112, "no recovery is pending for this account")
	ErrAlreadyApproved   = errors.Register(ModuleName, 1113, "this guardian has already approved")
	ErrRecoveryExpired   = errors.Register(ModuleName, 1114, "this recovery request has expired")
	ErrRecoveryLocked    = errors.Register(ModuleName, 1115, "recovery already has enough approvals; it can only be cancelled by the account's own key")
	ErrBadNewAddress     = errors.Register(ModuleName, 1116, "new address is invalid, or is the account being recovered")
	ErrNotAccountOwner   = errors.Register(ModuleName, 1117, "only the account's own key can do this")
)
