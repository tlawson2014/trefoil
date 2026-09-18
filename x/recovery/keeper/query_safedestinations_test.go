package keeper_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"trefoil/x/recovery/keeper"
	"trefoil/x/recovery/types"
)

func createNSafedestinations(keeper keeper.Keeper, ctx context.Context, n int) []types.Safedestinations {
	items := make([]types.Safedestinations, n)
	for i := range items {
		items[i].Owner = strconv.Itoa(i)
		items[i].Addresses = []string{`abc` + strconv.Itoa(i), `xyz` + strconv.Itoa(i)}
		_ = keeper.Safedestinations.Set(ctx, items[i].Owner, items[i])
	}
	return items
}

func TestSafedestinationsQuerySingle(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNSafedestinations(f.keeper, f.ctx, 2)
	tests := []struct {
		desc     string
		request  *types.QueryGetSafedestinationsRequest
		response *types.QueryGetSafedestinationsResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetSafedestinationsRequest{
				Owner: msgs[0].Owner,
			},
			response: &types.QueryGetSafedestinationsResponse{Safedestinations: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetSafedestinationsRequest{
				Owner: msgs[1].Owner,
			},
			response: &types.QueryGetSafedestinationsResponse{Safedestinations: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetSafedestinationsRequest{
				Owner: strconv.Itoa(100000),
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
			response, err := qs.GetSafedestinations(f.ctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.EqualExportedValues(t, tc.response, response)
			}
		})
	}
}

func TestSafedestinationsQueryPaginated(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNSafedestinations(f.keeper, f.ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllSafedestinationsRequest {
		return &types.QueryAllSafedestinationsRequest{
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
			resp, err := qs.ListSafedestinations(f.ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Safedestinations), step)
			require.Subset(t, msgs, resp.Safedestinations)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := qs.ListSafedestinations(f.ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Safedestinations), step)
			require.Subset(t, msgs, resp.Safedestinations)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := qs.ListSafedestinations(f.ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.EqualExportedValues(t, msgs, resp.Safedestinations)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := qs.ListSafedestinations(f.ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
