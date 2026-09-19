package keeper_test

import (
	"context"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"trefoil/x/undo/keeper"
	"trefoil/x/undo/types"
)

func createNPending(keeper keeper.Keeper, ctx context.Context, n int) []types.Pending {
	items := make([]types.Pending, n)
	for i := range items {
		iu := uint64(i)
		items[i].Id = iu
		items[i].Sender = strconv.Itoa(i)
		items[i].Recipient = strconv.Itoa(i)
		items[i].Amount = sdk.NewInt64Coin(`token`, int64(i+100))
		items[i].Executeat = int64(i)
		_ = keeper.Pending.Set(ctx, iu, items[i])
		_ = keeper.PendingSeq.Set(ctx, iu)
	}
	return items
}

func TestPendingQuerySingle(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNPending(f.keeper, f.ctx, 2)
	tests := []struct {
		desc     string
		request  *types.QueryGetPendingRequest
		response *types.QueryGetPendingResponse
		err      error
	}{
		{
			desc:     "First",
			request:  &types.QueryGetPendingRequest{Id: msgs[0].Id},
			response: &types.QueryGetPendingResponse{Pending: msgs[0]},
		},
		{
			desc:     "Second",
			request:  &types.QueryGetPendingRequest{Id: msgs[1].Id},
			response: &types.QueryGetPendingResponse{Pending: msgs[1]},
		},
		{
			desc:    "KeyNotFound",
			request: &types.QueryGetPendingRequest{Id: uint64(len(msgs))},
			err:     sdkerrors.ErrKeyNotFound,
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := qs.GetPending(f.ctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.EqualExportedValues(t, tc.response, response)
			}
		})
	}
}

func TestPendingQueryPaginated(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNPending(f.keeper, f.ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllPendingRequest {
		return &types.QueryAllPendingRequest{
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
			resp, err := qs.ListPending(f.ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Pending), step)
			require.Subset(t, msgs, resp.Pending)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := qs.ListPending(f.ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Pending), step)
			require.Subset(t, msgs, resp.Pending)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := qs.ListPending(f.ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.EqualExportedValues(t, msgs, resp.Pending)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := qs.ListPending(f.ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
