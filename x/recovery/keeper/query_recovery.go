package keeper

import (
	"context"
	"errors"

	"trefoil/x/recovery/types"

	"cosmossdk.io/collections"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q queryServer) ListRecovery(ctx context.Context, req *types.QueryAllRecoveryRequest) (*types.QueryAllRecoveryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	recoverys, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.Recovery,
		req.Pagination,
		func(_ string, value types.Recovery) (types.Recovery, error) {
			return value, nil
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllRecoveryResponse{Recovery: recoverys, Pagination: pageRes}, nil
}

func (q queryServer) GetRecovery(ctx context.Context, req *types.QueryGetRecoveryRequest) (*types.QueryGetRecoveryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	val, err := q.k.Recovery.Get(ctx, req.Account)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "not found")
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &types.QueryGetRecoveryResponse{Recovery: val}, nil
}
