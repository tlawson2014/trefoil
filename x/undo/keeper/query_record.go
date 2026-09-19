package keeper

import (
	"context"
	"errors"

	"trefoil/x/undo/types"

	"cosmossdk.io/collections"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q queryServer) ListRecord(ctx context.Context, req *types.QueryAllRecordRequest) (*types.QueryAllRecordResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	records, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.Record,
		req.Pagination,
		func(_ string, value types.Record) (types.Record, error) {
			return value, nil
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllRecordResponse{Record: records, Pagination: pageRes}, nil
}

func (q queryServer) GetRecord(ctx context.Context, req *types.QueryGetRecordRequest) (*types.QueryGetRecordResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	val, err := q.k.Record.Get(ctx, req.Index)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "not found")
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &types.QueryGetRecordResponse{Record: val}, nil
}
