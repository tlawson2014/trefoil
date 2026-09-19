package keeper_test

import (
	"testing"

	"trefoil/x/undo/types"

	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:       types.DefaultParams(),
		PendingList:  []types.Pending{{Id: 0}, {Id: 1}},
		PendingCount: 2,
		RecordMap:    []types.Record{{Index: "0"}, {Index: "1"}}}
	f := initFixture(t)
	err := f.keeper.InitGenesis(f.ctx, genesisState)
	require.NoError(t, err)
	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.NotNil(t, got)

	require.EqualExportedValues(t, genesisState.Params, got.Params)
	require.EqualExportedValues(t, genesisState.PendingList, got.PendingList)
	require.Equal(t, genesisState.PendingCount, got.PendingCount)
	require.EqualExportedValues(t, genesisState.RecordMap, got.RecordMap)

}
