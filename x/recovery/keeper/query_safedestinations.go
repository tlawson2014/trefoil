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

func (q queryServer) ListSafedestinations(ctx context.Context, req *types.QueryAllSafedestinationsRequest) (*types.QueryAllSafedestinationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	safedestinationss, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.Safedestinations,
		req.Pagination,
		func(_ string, value types.Safedestinations) (types.Safedestinations, error) {
			return value, nil
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllSafedestinationsResponse{Safedestinations: safedestinationss, Pagination: pageRes}, nil
}

func (q queryServer) GetSafedestinations(ctx context.Context, req *types.QueryGetSafedestinationsRequest) (*types.QueryGetSafedestinationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	val, err := q.k.Safedestinations.Get(ctx, req.Owner)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "not found")
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &types.QueryGetSafedestinationsResponse{Safedestinations: val}, nil
}
