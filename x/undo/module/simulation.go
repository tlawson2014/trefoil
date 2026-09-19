package undo

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	undosimulation "trefoil/x/undo/simulation"
	"trefoil/x/undo/types"
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	undoGenesis := types.GenesisState{
		Params: types.DefaultParams(),
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&undoGenesis)
}

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)
	const (
		opWeightMsgDelayedSend          = "op_weight_msg_undo"
		defaultWeightMsgDelayedSend int = 100
	)

	var weightMsgDelayedSend int
	simState.AppParams.GetOrGenerate(opWeightMsgDelayedSend, &weightMsgDelayedSend, nil,
		func(_ *rand.Rand) {
			weightMsgDelayedSend = defaultWeightMsgDelayedSend
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgDelayedSend,
		undosimulation.SimulateMsgDelayedSend(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgCancelSend          = "op_weight_msg_undo"
		defaultWeightMsgCancelSend int = 100
	)

	var weightMsgCancelSend int
	simState.AppParams.GetOrGenerate(opWeightMsgCancelSend, &weightMsgCancelSend, nil,
		func(_ *rand.Rand) {
			weightMsgCancelSend = defaultWeightMsgCancelSend
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCancelSend,
		undosimulation.SimulateMsgCancelSend(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgSetFinalOnly          = "op_weight_msg_undo"
		defaultWeightMsgSetFinalOnly int = 100
	)

	var weightMsgSetFinalOnly int
	simState.AppParams.GetOrGenerate(opWeightMsgSetFinalOnly, &weightMsgSetFinalOnly, nil,
		func(_ *rand.Rand) {
			weightMsgSetFinalOnly = defaultWeightMsgSetFinalOnly
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgSetFinalOnly,
		undosimulation.SimulateMsgSetFinalOnly(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{}
}
