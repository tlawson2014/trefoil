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

func (q queryServer) ListGuardianset(ctx context.Context, req *types.QueryAllGuardiansetRequest) (*types.QueryAllGuardiansetResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	guardiansets, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.Guardianset,
		req.Pagination,
		func(_ string, value types.Guardianset) (types.Guardianset, error) {
			return value, nil
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllGuardiansetResponse{Guardianset: guardiansets, Pagination: pageRes}, nil
}

func (q queryServer) GetGuardianset(ctx context.Context, req *types.QueryGetGuardiansetRequest) (*types.QueryGetGuardiansetResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	val, err := q.k.Guardianset.Get(ctx, req.Owner)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "not found")
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &types.QueryGetGuardiansetResponse{Guardianset: val}, nil
}
