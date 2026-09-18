package keeper

import (
	"context"

	"trefoil/x/recovery/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	for _, elem := range genState.GuardiansetMap {
		if err := k.Guardianset.Set(ctx, elem.Owner, elem); err != nil {
			return err
		}
	}
	for _, elem := range genState.RecoveryMap {
		if err := k.Recovery.Set(ctx, elem.Account, elem); err != nil {
			return err
		}
	}
	for _, elem := range genState.SafedestinationsMap {
		if err := k.Safedestinations.Set(ctx, elem.Owner, elem); err != nil {
			return err
		}
	}

	return k.Params.Set(ctx, genState.Params)
}

// ExportGenesis returns the module's exported genesis.
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	var err error

	genesis := types.DefaultGenesis()
	genesis.Params, err = k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	if err := k.Guardianset.Walk(ctx, nil, func(_ string, val types.Guardianset) (stop bool, err error) {
		genesis.GuardiansetMap = append(genesis.GuardiansetMap, val)
		return false, nil
	}); err != nil {
		return nil, err
	}
	if err := k.Recovery.Walk(ctx, nil, func(_ string, val types.Recovery) (stop bool, err error) {
		genesis.RecoveryMap = append(genesis.RecoveryMap, val)
		return false, nil
	}); err != nil {
		return nil, err
	}
	if err := k.Safedestinations.Walk(ctx, nil, func(_ string, val types.Safedestinations) (stop bool, err error) {
		genesis.SafedestinationsMap = append(genesis.SafedestinationsMap, val)
		return false, nil
	}); err != nil {
		return nil, err
	}

	return genesis, nil
}
