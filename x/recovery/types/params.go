package types

import "fmt"

// Module parameters. These are the two dials from the design doc. They live
// on-chain, so governance can change them later without a software upgrade.
//
// Both are in SECONDS.
//
//	Recoverydelay — once enough guardians have approved, how long the chain
//	                waits before actually executing the recovery. This is the
//	                window in which the account's original key can cancel.
//	                Design doc: 48 hours. Local testing: 60 seconds
//	                (overridden in config.yml).
//
//	Requestexpiry — how long an unapproved request stays open before the
//	                chain forgets it. Design doc: 7 days.
const (
	DefaultRecoverydelay int64 = 48 * 60 * 60     // 172800
	DefaultRequestexpiry int64 = 7 * 24 * 60 * 60 // 604800
)

// NewParams creates a new Params instance.
func NewParams(recoverydelay int64, requestexpiry int64) Params {
	return Params{
		Recoverydelay: recoverydelay,
		Requestexpiry: requestexpiry,
	}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams(DefaultRecoverydelay, DefaultRequestexpiry)
}

// Validate validates the set of params.
func (p Params) Validate() error {
	if err := validateRecoverydelay(p.Recoverydelay); err != nil {
		return err
	}
	if err := validateRequestexpiry(p.Requestexpiry); err != nil {
		return err
	}
	return nil
}

func validateRecoverydelay(v int64) error {
	if v <= 0 {
		return fmt.Errorf("recoverydelay must be a positive number of seconds, got %d", v)
	}
	return nil
}

func validateRequestexpiry(v int64) error {
	if v <= 0 {
		return fmt.Errorf("requestexpiry must be a positive number of seconds, got %d", v)
	}
	return nil
}
