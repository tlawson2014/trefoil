package recovery

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	recoverysimulation "trefoil/x/recovery/simulation"
	"trefoil/x/recovery/types"
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	recoveryGenesis := types.GenesisState{
		Params: types.DefaultParams(),
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&recoveryGenesis)
}

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)
	const (
		opWeightMsgSetGuardians          = "op_weight_msg_recovery"
		defaultWeightMsgSetGuardians int = 100
	)

	var weightMsgSetGuardians int
	simState.AppParams.GetOrGenerate(opWeightMsgSetGuardians, &weightMsgSetGuardians, nil,
		func(_ *rand.Rand) {
			weightMsgSetGuardians = defaultWeightMsgSetGuardians
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgSetGuardians,
		recoverysimulation.SimulateMsgSetGuardians(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgRequestRecovery          = "op_weight_msg_recovery"
		defaultWeightMsgRequestRecovery int = 100
	)

	var weightMsgRequestRecovery int
	simState.AppParams.GetOrGenerate(opWeightMsgRequestRecovery, &weightMsgRequestRecovery, nil,
		func(_ *rand.Rand) {
			weightMsgRequestRecovery = defaultWeightMsgRequestRecovery
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgRequestRecovery,
		recoverysimulation.SimulateMsgRequestRecovery(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgApproveRecovery          = "op_weight_msg_recovery"
		defaultWeightMsgApproveRecovery int = 100
	)

	var weightMsgApproveRecovery int
	simState.AppParams.GetOrGenerate(opWeightMsgApproveRecovery, &weightMsgApproveRecovery, nil,
		func(_ *rand.Rand) {
			weightMsgApproveRecovery = defaultWeightMsgApproveRecovery
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgApproveRecovery,
		recoverysimulation.SimulateMsgApproveRecovery(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgCancelRecovery          = "op_weight_msg_recovery"
		defaultWeightMsgCancelRecovery int = 100
	)

	var weightMsgCancelRecovery int
	simState.AppParams.GetOrGenerate(opWeightMsgCancelRecovery, &weightMsgCancelRecovery, nil,
		func(_ *rand.Rand) {
			weightMsgCancelRecovery = defaultWeightMsgCancelRecovery
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCancelRecovery,
		recoverysimulation.SimulateMsgCancelRecovery(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgSetSafeDestinations          = "op_weight_msg_recovery"
		defaultWeightMsgSetSafeDestinations int = 100
	)

	var weightMsgSetSafeDestinations int
	simState.AppParams.GetOrGenerate(opWeightMsgSetSafeDestinations, &weightMsgSetSafeDestinations, nil,
		func(_ *rand.Rand) {
			weightMsgSetSafeDestinations = defaultWeightMsgSetSafeDestinations
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgSetSafeDestinations,
		recoverysimulation.SimulateMsgSetSafeDestinations(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{}
}
