package keeper_test

import (
	"testing"

	"trefoil/x/recovery/types"

	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:         types.DefaultParams(),
		GuardiansetMap: []types.Guardianset{{Owner: "0"}, {Owner: "1"}}, RecoveryMap: []types.Recovery{{Account: "0"}, {Account: "1"}}}

	f := initFixture(t)
	err := f.keeper.InitGenesis(f.ctx, genesisState)
	require.NoError(t, err)
	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.NotNil(t, got)

	require.EqualExportedValues(t, genesisState.Params, got.Params)
	require.EqualExportedValues(t, genesisState.GuardiansetMap, got.GuardiansetMap)
	require.EqualExportedValues(t, genesisState.RecoveryMap, got.RecoveryMap)

}
