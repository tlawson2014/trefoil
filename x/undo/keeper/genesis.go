package keeper

import (
	"context"

	"trefoil/x/undo/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	for _, elem := range genState.PendingList {
		if err := k.Pending.Set(ctx, elem.Id, elem); err != nil {
			return err
		}
	}

	if err := k.PendingSeq.Set(ctx, genState.PendingCount); err != nil {
		return err
	}
	for _, elem := range genState.RecordMap {
		if err := k.Record.Set(ctx, elem.Index, elem); err != nil {
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
	err = k.Pending.Walk(ctx, nil, func(key uint64, elem types.Pending) (bool, error) {
		genesis.PendingList = append(genesis.PendingList, elem)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	genesis.PendingCount, err = k.PendingSeq.Peek(ctx)
	if err != nil {
		return nil, err
	}
	if err := k.Record.Walk(ctx, nil, func(_ string, val types.Record) (stop bool, err error) {
		genesis.RecordMap = append(genesis.RecordMap, val)
		return false, nil
	}); err != nil {
		return nil, err
	}

	return genesis, nil
}
