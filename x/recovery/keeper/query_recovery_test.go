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

func createNRecovery(keeper keeper.Keeper, ctx context.Context, n int) []types.Recovery {
	items := make([]types.Recovery, n)
	for i := range items {
		items[i].Account = strconv.Itoa(i)
		items[i].Newaddress = strconv.Itoa(i)
		items[i].Requester = strconv.Itoa(i)
		items[i].Approvals = []string{`abc` + strconv.Itoa(i), `xyz` + strconv.Itoa(i)}
		items[i].Requestedat = int64(i)
		items[i].Executeat = int64(i)
		_ = keeper.Recovery.Set(ctx, items[i].Account, items[i])
	}
	return items
}

func TestRecoveryQuerySingle(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNRecovery(f.keeper, f.ctx, 2)
	tests := []struct {
		desc     string
		request  *types.QueryGetRecoveryRequest
		response *types.QueryGetRecoveryResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetRecoveryRequest{
				Account: msgs[0].Account,
			},
			response: &types.QueryGetRecoveryResponse{Recovery: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetRecoveryRequest{
				Account: msgs[1].Account,
			},
			response: &types.QueryGetRecoveryResponse{Recovery: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetRecoveryRequest{
				Account: strconv.Itoa(100000),
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
			response, err := qs.GetRecovery(f.ctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.EqualExportedValues(t, tc.response, response)
			}
		})
	}
}

func TestRecoveryQueryPaginated(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNRecovery(f.keeper, f.ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllRecoveryRequest {
		return &types.QueryAllRecoveryRequest{
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
			resp, err := qs.ListRecovery(f.ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Recovery), step)
			require.Subset(t, msgs, resp.Recovery)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := qs.ListRecovery(f.ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Recovery), step)
			require.Subset(t, msgs, resp.Recovery)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := qs.ListRecovery(f.ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.EqualExportedValues(t, msgs, resp.Recovery)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := qs.ListRecovery(f.ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
