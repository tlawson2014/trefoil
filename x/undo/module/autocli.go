package undo

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	"trefoil/x/undo/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: types.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				{
					RpcMethod: "ListPending",
					Use:       "list-pending",
					Short:     "List all pending",
				},
				{
					RpcMethod:      "GetPending",
					Use:            "get-pending [id]",
					Short:          "Gets a pending by id",
					Alias:          []string{"show-pending"},
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{
					RpcMethod: "ListRecord",
					Use:       "list-record",
					Short:     "List all record",
				},
				{
					RpcMethod:      "GetRecord",
					Use:            "get-record [id]",
					Short:          "Gets a record",
					Alias:          []string{"show-record"},
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "index"}},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              types.Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod:      "DelayedSend",
					Use:            "delayed-send [recipient] [amount] [window]",
					Short:          "Send a delayed-send tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "recipient"}, {ProtoField: "amount"}, {ProtoField: "window"}},
				},
				{
					RpcMethod:      "CancelSend",
					Use:            "cancel-send [id]",
					Short:          "Send a cancel-send tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{
					RpcMethod:      "SetFinalOnly",
					Use:            "set-final-only [finalonly]",
					Short:          "Send a set-final-only tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "finalonly"}},
				},
			},
		},
	}
}
