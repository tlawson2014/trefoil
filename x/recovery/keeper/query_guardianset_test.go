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

func createNGuardianset(keeper keeper.Keeper, ctx context.Context, n int) []types.Guardianset {
	items := make([]types.Guardianset, n)
	for i := range items {
		items[i].Owner = strconv.Itoa(i)
		items[i].Threshold = uint64(i)
		items[i].Guardians = []string{`abc` + strconv.Itoa(i), `xyz` + strconv.Itoa(i)}
		_ = keeper.Guardianset.Set(ctx, items[i].Owner, items[i])
	}
	return items
}

func TestGuardiansetQuerySingle(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNGuardianset(f.keeper, f.ctx, 2)
	tests := []struct {
		desc     string
		request  *types.QueryGetGuardiansetRequest
		response *types.QueryGetGuardiansetResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetGuardiansetRequest{
				Owner: msgs[0].Owner,
			},
			response: &types.QueryGetGuardiansetResponse{Guardianset: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetGuardiansetRequest{
				Owner: msgs[1].Owner,
			},
			response: &types.QueryGetGuardiansetResponse{Guardianset: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetGuardiansetRequest{
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
			response, err := qs.GetGuardianset(f.ctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.EqualExportedValues(t, tc.response, response)
			}
		})
	}
}

func TestGuardiansetQueryPaginated(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNGuardianset(f.keeper, f.ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllGuardiansetRequest {
		return &types.QueryAllGuardiansetRequest{
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
			resp, err := qs.ListGuardianset(f.ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Guardianset), step)
			require.Subset(t, msgs, resp.Guardianset)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := qs.ListGuardianset(f.ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Guardianset), step)
			require.Subset(t, msgs, resp.Guardianset)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := qs.ListGuardianset(f.ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.EqualExportedValues(t, msgs, resp.Guardianset)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := qs.ListGuardianset(f.ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
