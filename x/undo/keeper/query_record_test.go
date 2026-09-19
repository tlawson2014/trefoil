package keeper_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"trefoil/x/undo/keeper"
	"trefoil/x/undo/types"
)

func createNRecord(keeper keeper.Keeper, ctx context.Context, n int) []types.Record {
	items := make([]types.Record, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)
		items[i].Sent = uint64(i)
		items[i].Cancelled = uint64(i)
		_ = keeper.Record.Set(ctx, items[i].Index, items[i])
	}
	return items
}

func TestRecordQuerySingle(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNRecord(f.keeper, f.ctx, 2)
	tests := []struct {
		desc     string
		request  *types.QueryGetRecordRequest
		response *types.QueryGetRecordResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetRecordRequest{
				Index: msgs[0].Index,
			},
			response: &types.QueryGetRecordResponse{Record: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetRecordRequest{
				Index: msgs[1].Index,
			},
			response: &types.QueryGetRecordResponse{Record: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetRecordRequest{
				Index: strconv.Itoa(100000),
			},
			err: status.Error(codes.NotFound, "not found"),
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := qs.GetRecord(f.ctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.EqualExportedValues(t, tc.response, response)
			}
		})
	}
}

func TestRecordQueryPaginated(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNRecord(f.keeper, f.ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllRecordRequest {
		return &types.QueryAllRecordRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}
	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(msgs); i += step {
			resp, err := qs.ListRecord(f.ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Record), step)
			require.Subset(t, msgs, resp.Record)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := qs.ListRecord(f.ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Record), step)
			require.Subset(t, msgs, resp.Record)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := qs.ListRecord(f.ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.EqualExportedValues(t, msgs, resp.Record)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := qs.ListRecord(f.ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
