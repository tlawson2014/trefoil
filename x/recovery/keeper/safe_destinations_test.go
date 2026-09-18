package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trefoil/x/recovery/types"
)

// Safe destinations: the answer to "what if two guardians are compromised?"
// Once set, a recovery can only send funds to an address the owner chose in
// advance, so colluding guardians gain nothing.

var (
	backup  = addr(3) // Tom's spare key in a drawer
	backup2 = addr(4) // a second spare
)

func (h *harness) setSafe(owner string, addrs []string) error {
	_, err := h.msg.SetSafeDestinations(h.ctx, &types.MsgSetSafeDestinations{Creator: owner, Addresses: addrs})
	return err
}

// ---------------------------------------------------------------------------
// Setting safe destinations
// ---------------------------------------------------------------------------

func TestSafeDestinations_Set(t *testing.T) {
	h := newHarness(t)
	require.NoError(t, h.setSafe(tom, []string{backup, backup2}))

	sd, err := h.k.Safedestinations.Get(h.ctx, tom)
	require.NoError(t, err)
	require.Equal(t, tom, sd.Owner)
	require.Equal(t, []string{backup, backup2}, sd.Addresses)

	// Setting an empty list clears the restriction entirely.
	require.NoError(t, h.setSafe(tom, nil))
	_, err = h.k.Safedestinations.Get(h.ctx, tom)
	require.Error(t, err, "empty list should remove the record")

	// Clearing when nothing is set is harmless.
	require.NoError(t, h.setSafe(tom, nil))
}

func TestSafeDestinations_Rules(t *testing.T) {
	h := newHarness(t)

	require.ErrorIs(t, h.setSafe(tom, []string{backup, backup}), types.ErrBadSafeDestination) // duplicate
	require.ErrorIs(t, h.setSafe(tom, []string{tom}), types.ErrBadSafeDestination)            // self
	require.Error(t, h.setSafe(tom, []string{"garbage"}))                                     // invalid address

	six := make([]string, 6)
	for i := range six {
		six[i] = addr(byte(30 + i))
	}
	require.ErrorIs(t, h.setSafe(tom, six), types.ErrTooManySafeDestinations)
}

func TestSafeDestinations_BlockedWhileRecoveryPending(t *testing.T) {
	h := newHarness(t)
	require.NoError(t, h.setGuardians(tom, threeG, 2))
	require.NoError(t, h.setSafe(tom, []string{backup}))
	require.NoError(t, h.request(g1, tom, backup))

	// A thief with Tom's key cannot point the recovery at themselves mid-flight.
	require.ErrorIs(t, h.setSafe(tom, []string{rando}), types.ErrRecoveryPending)
	require.ErrorIs(t, h.setSafe(tom, nil), types.ErrRecoveryPending) // nor clear the list
}

// ---------------------------------------------------------------------------
// The point of the feature: colluding guardians get nowhere
// ---------------------------------------------------------------------------

func TestSafeDestinations_GuardiansCannotRecoverElsewhere(t *testing.T) {
	h := newHarness(t)
	h.fund(tom, 1_000_000)
	require.NoError(t, h.setGuardians(tom, threeG, 2))
	require.NoError(t, h.setSafe(tom, []string{backup}))

	// Two guardians try to recover Tom's coins into a wallet they control.
	// The request is refused at the door — it never even reaches approval.
	require.ErrorIs(t, h.request(g1, tom, rando), types.ErrNotSafeDestination)
	require.ErrorIs(t, h.request(g1, tom, g1), types.ErrNotSafeDestination)

	_, err := h.k.Recovery.Get(h.ctx, tom)
	require.Error(t, err, "no recovery should exist")
	require.Equal(t, int64(1_000_000), h.balance(tom))
}

func TestSafeDestinations_RecoveryToBackupWorks(t *testing.T) {
	h := newHarness(t)
	h.fund(tom, 1_000_000)
	require.NoError(t, h.setGuardians(tom, threeG, 2))
	require.NoError(t, h.setSafe(tom, []string{backup, backup2}))

	// Recovery to a registered backup goes through normally.
	require.NoError(t, h.request(g1, tom, backup))
	require.NoError(t, h.approve(g2, tom))
	h.advance(61)
	h.endBlock()

	require.Equal(t, int64(1_000_000), h.balance(backup))
	require.Equal(t, int64(0), h.balance(tom))

	// The safe-destination list followed Tom to the backup address, minus
	// the backup itself (an account is never its own destination).
	sd, err := h.k.Safedestinations.Get(h.ctx, backup)
	require.NoError(t, err)
	require.Equal(t, backup, sd.Owner)
	require.Equal(t, []string{backup2}, sd.Addresses)

	_, err = h.k.Safedestinations.Get(h.ctx, tom)
	require.Error(t, err, "old account's list should be gone")
}

func TestSafeDestinations_SingleBackupListDropsAfterUse(t *testing.T) {
	h := newHarness(t)
	h.fund(tom, 10)
	require.NoError(t, h.setGuardians(tom, threeG, 2))
	require.NoError(t, h.setSafe(tom, []string{backup}))

	require.NoError(t, h.request(g1, tom, backup))
	require.NoError(t, h.approve(g2, tom))
	h.advance(61)
	h.endBlock()

	// Only destination was the backup we recovered into, so the new account
	// has no restriction until the owner sets a fresh one.
	_, err := h.k.Safedestinations.Get(h.ctx, backup)
	require.Error(t, err)
	require.Equal(t, int64(10), h.balance(backup))
}

func TestSafeDestinations_NoneSetMeansNoRestriction(t *testing.T) {
	h := newHarness(t)
	require.NoError(t, h.setGuardians(tom, threeG, 2))

	// v1 behaviour is unchanged for accounts that never set a list.
	require.NoError(t, h.request(g1, tom, rando))
}
