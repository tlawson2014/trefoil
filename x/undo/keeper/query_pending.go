package keeper

import (
	"context"
	"errors"

	"trefoil/x/undo/types"

	"cosmossdk.io/collections"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q queryServer) ListPending(ctx context.Context, req *types.QueryAllPendingRequest) (*types.QueryAllPendingResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	pendings, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.Pending,
		req.Pagination,
		func(_ uint64, value types.Pending) (types.Pending, error) {
			return value, nil
		},
	)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllPendingResponse{Pending: pendings, Pagination: pageRes}, nil
}

func (q queryServer) GetPending(ctx context.Context, req *types.QueryGetPendingRequest) (*types.QueryGetPendingResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	pending, err := q.k.Pending.Get(ctx, req.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, sdkerrors.ErrKeyNotFound
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &types.QueryGetPendingResponse{Pending: pending}, nil
}
