package types

import (
	"context"

	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// These interfaces list the ONLY things the recovery module is allowed to ask
// other modules to do. Keeping them small is a Cosmos habit worth copying: it
// makes it obvious, at a glance, what power this module has over people's
// accounts and money.

// AuthKeeper defines the expected interface for the Auth (accounts) module.
type AuthKeeper interface {
	AddressCodec() address.Codec
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI
}

// BankKeeper defines the expected interface for the Bank module.
//
// GetAllBalances + SendCoins are what a recovery uses to move everything the
// lost account holds to the new address. Nothing else — the module cannot
// mint, burn, or touch any account that isn't mid-recovery.
type BankKeeper interface {
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
	GetAllBalances(context.Context, sdk.AccAddress) sdk.Coins
	SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}
