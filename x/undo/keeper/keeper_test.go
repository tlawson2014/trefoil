package keeper_test

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/store/types"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"trefoil/x/undo/keeper"
	module "trefoil/x/undo/module"
	"trefoil/x/undo/types"
)

// mockBank is a toy bank for tests: a map of balances plus one pot for the
// undo module's holding account. It refuses to overdraw, which is the only
// bank behaviour the undo module relies on.
type mockBank struct {
	balances map[string]sdk.Coins
	holding  sdk.Coins
}

func newMockBank() *mockBank { return &mockBank{balances: map[string]sdk.Coins{}} }

func key(a sdk.AccAddress) string { return hex.EncodeToString(a) }

func (m *mockBank) SpendableCoins(_ context.Context, a sdk.AccAddress) sdk.Coins {
	return m.balances[key(a)]
}

func (m *mockBank) SendCoins(_ context.Context, from, to sdk.AccAddress, amt sdk.Coins) error {
	if !m.balances[key(from)].IsAllGTE(amt) {
		return errors.New("insufficient funds")
	}
	m.balances[key(from)] = m.balances[key(from)].Sub(amt...)
	m.balances[key(to)] = m.balances[key(to)].Add(amt...)
	return nil
}

func (m *mockBank) SendCoinsFromAccountToModule(_ context.Context, from sdk.AccAddress, _ string, amt sdk.Coins) error {
	if !m.balances[key(from)].IsAllGTE(amt) {
		return errors.New("insufficient funds")
	}
	m.balances[key(from)] = m.balances[key(from)].Sub(amt...)
	m.holding = m.holding.Add(amt...)
	return nil
}

func (m *mockBank) SendCoinsFromModuleToAccount(_ context.Context, _ string, to sdk.AccAddress, amt sdk.Coins) error {
	if !m.holding.IsAllGTE(amt) {
		return errors.New("holding account short")
	}
	m.holding = m.holding.Sub(amt...)
	m.balances[key(to)] = m.balances[key(to)].Add(amt...)
	return nil
}

type fixture struct {
	ctx          sdk.Context
	keeper       keeper.Keeper
	addressCodec address.Codec
	bank         *mockBank
}

func initFixture(t *testing.T) *fixture {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx
	ctx = ctx.WithBlockTime(time.Unix(1_000_000, 0))

	authority := authtypes.NewModuleAddress(types.GovModuleName)
	bank := newMockBank()

	k := keeper.NewKeeper(
		storeService,
		encCfg.Codec,
		addressCodec,
		authority,
		bank,
		nil,
	)

	// Initialize params
	if err := k.Params.Set(ctx, types.DefaultParams()); err != nil {
		t.Fatalf("failed to set params: %v", err)
	}

	return &fixture{
		ctx:          ctx,
		keeper:       k,
		addressCodec: addressCodec,
		bank:         bank,
	}
}
