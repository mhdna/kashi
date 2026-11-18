package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func TestReferralsOnAt(t *testing.T) {
	on := db.ReferralProgram{Enabled: true, BranchIds: []int64{7, 9}}
	require.True(t, referralsOnAt(on, 7))
	require.True(t, referralsOnAt(on, 9))
	require.False(t, referralsOnAt(on, 8), "a branch not ticked is off")
	require.False(t, referralsOnAt(db.ReferralProgram{Enabled: false, BranchIds: []int64{7}}, 7), "the program off is off everywhere")
	require.False(t, referralsOnAt(db.ReferralProgram{Enabled: true}, 7), "on with no branches is on nowhere")
}

func TestReferralProgramRequestBranchIDs(t *testing.T) {
	all := []int64{3, 7, 9}
	ids := func(v ...int64) *[]int64 { return &v }
	require.Equal(t, all, referralProgramRequest{Enabled: true}.branchIDs(all), "an older page saying on means every branch")
	require.Equal(t, []int64{}, referralProgramRequest{Enabled: false}.branchIDs(all))
	require.Equal(t, []int64{3, 9}, referralProgramRequest{BranchIDs: ids(9, 3, 9, 0, -2)}.branchIDs(all), "each once, in order, no nonsense ids")
	require.Equal(t, []int64{}, referralProgramRequest{Enabled: true, BranchIDs: ids()}.branchIDs(all), "an empty list is off, whatever enabled says")
}

func referralRules(extra map[string]any) map[string]any {
	body := map[string]any{"reward_percent": 15, "cancel_on_return": true}
	for k, v := range extra {
		body[k] = v
	}
	return body
}

// The reported case: referrals on at Centro only.
func TestUpdateReferralProgramPerBranch(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ListBranches(gomock.Any()).Return([]db.Branch{{ID: 7, Name: "Centro"}, {ID: 9, Name: "Hara"}}, nil)
	store.EXPECT().UpdateReferralProgram(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.UpdateReferralProgramParams) (db.ReferralProgram, error) {
			require.Equal(t, []int64{7}, arg.BranchIds)
			require.True(t, arg.Enabled)
			return db.ReferralProgram{Enabled: true, BranchIds: arg.BranchIds, RewardPercent: 15}, nil
		})
	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPut, "/referral_program", referralRules(map[string]any{"branch_ids": []int64{7}}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), `"branch_ids":[7]`)
}

func TestUpdateReferralProgramNoBranchesIsOff(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ListBranches(gomock.Any()).Return([]db.Branch{{ID: 7}}, nil)
	store.EXPECT().UpdateReferralProgram(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.UpdateReferralProgramParams) (db.ReferralProgram, error) {
			require.Equal(t, []int64{}, arg.BranchIds)
			require.False(t, arg.Enabled)
			return db.ReferralProgram{}, nil
		})
	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPut, "/referral_program", referralRules(map[string]any{"enabled": true, "branch_ids": []int64{}}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestUpdateReferralProgramUnknownBranch(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ListBranches(gomock.Any()).Return([]db.Branch{{ID: 7}}, nil)
	store.EXPECT().UpdateReferralProgram(gomock.Any(), gomock.Any()).Times(0)
	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPut, "/referral_program", referralRules(map[string]any{"branch_ids": []int64{7, 44}}))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "no branch #44")
}

// Each branch's tills are told whether referrals are on at their branch.
func TestReferralProgramFeedPerBranch(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	program := db.ReferralProgram{Enabled: true, BranchIds: []int64{7}, RewardPercent: 20, MinFriendSpend: 1000, UpdatedAt: time.Now()}
	store.EXPECT().GetReferralProgramUpdatedSince(gomock.Any(), gomock.Any()).Return([]db.ReferralProgram{program}, nil).Times(2)
	server := newTestServer(t, store)

	centro, err := server.referralProgramForBranchSync(context.Background(), 7, time.Time{})
	require.NoError(t, err)
	require.Equal(t, []syncReferralProgram{{Enabled: true, RewardPercent: 20, MinFriendSpend: 1000, UpdatedAt: program.UpdatedAt}}, centro)
	hara, err := server.referralProgramForBranchSync(context.Background(), 9, time.Time{})
	require.NoError(t, err)
	require.False(t, hara[0].Enabled)
	require.EqualValues(t, 20, hara[0].RewardPercent, "the rules are the same everywhere")
}
