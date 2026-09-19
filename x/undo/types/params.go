package types

import "fmt"

// The three dials of the undo module, all in seconds.
//
//	Minwindow      the shortest undo window any send can have. Every send
//	               on Trefoil waits at least this long; there are no
//	               instant payments except to final-only addresses.
//	Defaultwindow  used when a sender doesn't pick a window (window = 0).
//	Maxwindow      the longest a sender may keep a payment in limbo.
//
// Defaults: 2 minutes / 10 minutes / 24 hours.
var (
	DefaultMinwindow     uint64 = 120
	DefaultDefaultwindow uint64 = 600
	DefaultMaxwindow     uint64 = 86400
)

// MaxOpenPerSender caps how many undoable sends one address can have in
// flight at once. Without it, one account could flood someone with fake
// "incoming" payments it intends to cancel.
const MaxOpenPerSender = 20

// NewParams creates a new Params instance.
func NewParams(minwindow, defaultwindow, maxwindow uint64) Params {
	return Params{Minwindow: minwindow, Defaultwindow: defaultwindow, Maxwindow: maxwindow}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams(DefaultMinwindow, DefaultDefaultwindow, DefaultMaxwindow)
}

// Validate validates the set of params.
func (p Params) Validate() error {
	if p.Minwindow == 0 {
		return fmt.Errorf("minwindow must be a positive number of seconds")
	}
	if p.Defaultwindow < p.Minwindow {
		return fmt.Errorf("defaultwindow (%d) must be at least minwindow (%d)", p.Defaultwindow, p.Minwindow)
	}
	if p.Maxwindow < p.Defaultwindow {
		return fmt.Errorf("maxwindow (%d) must be at least defaultwindow (%d)", p.Maxwindow, p.Defaultwindow)
	}
	return nil
}
