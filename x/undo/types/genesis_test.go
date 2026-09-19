package types_test

import (
	"testing"

	"trefoil/x/undo/types"

	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				PendingList:  []types.Pending{{Id: 0}, {Id: 1}},
				PendingCount: 2,
				RecordMap:    []types.Record{{Index: "0"}, {Index: "1"}},
			},
			valid: true,
		},
		{
			desc: "zero params are rejected",
			genState: &types.GenesisState{
				PendingList:  []types.Pending{},
				PendingCount: 0,
				RecordMap:    []types.Record{},
			},
			valid: false,
		},
		{
			desc: "duplicated pending",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				PendingList:  []types.Pending{{Id: 0}, {Id: 0}},
				PendingCount: 1,
				RecordMap:    []types.Record{{Index: "0"}, {Index: "1"}},
			},
			valid: false,
		},
		{
			desc: "invalid pending count",
			genState: &types.GenesisState{
				Params:       types.DefaultParams(),
				PendingList:  []types.Pending{{Id: 1}},
				PendingCount: 0,
				RecordMap:    []types.Record{{Index: "0"}, {Index: "1"}},
			},
			valid: false,
		},
		{
			desc: "duplicated record",
			genState: &types.GenesisState{
				Params:    types.DefaultParams(),
				RecordMap: []types.Record{{Index: "0"}, {Index: "0"}},
			},
			valid: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
