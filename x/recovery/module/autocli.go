package recovery

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	"trefoil/x/recovery/types"
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
					RpcMethod: "ListGuardianset",
					Use:       "list-guardianset",
					Short:     "List all guardianset",
				},
				{
					RpcMethod:      "GetGuardianset",
					Use:            "get-guardianset [id]",
					Short:          "Gets a guardianset",
					Alias:          []string{"show-guardianset"},
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "owner"}},
				},
				{
					RpcMethod: "ListRecovery",
					Use:       "list-recovery",
					Short:     "List all recovery",
				},
				{
					RpcMethod:      "GetRecovery",
					Use:            "get-recovery [id]",
					Short:          "Gets a recovery",
					Alias:          []string{"show-recovery"},
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "account"}},
				},
				{
					RpcMethod: "ListSafedestinations",
					Use:       "list-safedestinations",
					Short:     "List all safedestinations",
				},
				{
					RpcMethod:      "GetSafedestinations",
					Use:            "get-safedestinations [id]",
					Short:          "Gets a safedestinations",
					Alias:          []string{"show-safedestinations"},
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "owner"}},
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
					RpcMethod:      "SetGuardians",
					Use:            "set-guardians [guardians] [threshold]",
					Short:          "Send a set-guardians tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "guardians"}, {ProtoField: "threshold"}},
				},
				{
					RpcMethod:      "RequestRecovery",
					Use:            "request-recovery [account] [newaddress]",
					Short:          "Send a request-recovery tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "account"}, {ProtoField: "newaddress"}},
				},
				{
					RpcMethod:      "ApproveRecovery",
					Use:            "approve-recovery [account]",
					Short:          "Send a approve-recovery tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "account"}},
				},
				{
					RpcMethod:      "CancelRecovery",
					Use:            "cancel-recovery [account]",
					Short:          "Send a cancel-recovery tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "account"}},
				},
				{
					RpcMethod:      "SetSafeDestinations",
					Use:            "set-safe-destinations [addresses]",
					Short:          "Send a set-safe-destinations tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "addresses", Varargs: true}},
				},
			},
		},
	}
}
