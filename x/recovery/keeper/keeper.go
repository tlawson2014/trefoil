package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"

	"trefoil/x/recovery/types"
)

type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec
	// Address capable of executing a MsgUpdateParams message.
	// Typically, this should be the x/gov module account.
	authority []byte

	Schema collections.Schema
	Params collections.Item[types.Params]

	authKeeper       types.AuthKeeper
	bankKeeper       types.BankKeeper
	Guardianset      collections.Map[string, types.Guardianset]
	Recovery         collections.Map[string, types.Recovery]
	Safedestinations collections.Map[string, types.Safedestinations]
}

func NewKeeper(
	storeService corestore.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,
	authority []byte,

	authKeeper types.AuthKeeper,
	bankKeeper types.BankKeeper,
) Keeper {
	if _, err := addressCodec.BytesToString(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address %s: %s", authority, err))
	}

	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		storeService: storeService,
		cdc:          cdc,
		addressCodec: addressCodec,
		authority:    authority,

		authKeeper:  authKeeper,
		bankKeeper:  bankKeeper,
		Params:      collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		Guardianset: collections.NewMap(sb, types.GuardiansetKey, "guardianset", collections.StringKey, codec.CollValue[types.Guardianset](cdc)), Recovery: collections.NewMap(sb, types.RecoveryKey, "recovery", collections.StringKey, codec.CollValue[types.Recovery](cdc)), Safedestinations: collections.NewMap(sb, types.SafedestinationsKey, "safedestinations", collections.StringKey, codec.CollValue[types.Safedestinations](cdc))}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() []byte {
	return k.authority
}
