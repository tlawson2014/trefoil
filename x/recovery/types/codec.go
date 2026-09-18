package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterInterfaces(registrar codectypes.InterfaceRegistry) {
	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSetSafeDestinations{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCancelRecovery{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgApproveRecovery{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRequestRecovery{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSetGuardians{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
	)
	msgservice.RegisterMsgServiceDesc(registrar, &_Msg_serviceDesc)
}
