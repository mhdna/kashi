package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

const (
	testBranchCode   = "BR01"
	testBranchKey    = "branch-secret-key"
	testBranchID     = int64(7)
	testRegisterID   = int64(11)
	testRegisterCode = testBranchCode
)

// stubBranchAuth wires the mock so branchAuthMiddleware resolves
// testBranchCode/testBranchKey to an active register (testRegisterID) of an
// active branch (testBranchID).
func stubBranchAuth(store *mockdb.MockStore) {
	hash, _ := util.HashPassword(testBranchKey)
	store.EXPECT().
		GetBranchRegisterByCode(gomock.Any(), gomock.Eq(testRegisterCode)).
		Return(db.BranchRegister{ID: testRegisterID, BranchID: testBranchID, Code: testRegisterCode, ApiKeyHash: hash, IsActive: true}, nil).
		AnyTimes()
	store.EXPECT().
		GetBranch(gomock.Any(), gomock.Eq(testBranchID)).
		Return(db.Branch{ID: testBranchID, Code: testBranchCode, IsActive: true}, nil).
		AnyTimes()
	store.EXPECT().
		GetBranchRegister(gomock.Any(), gomock.Eq(testRegisterID)).
		Return(db.BranchRegister{ID: testRegisterID, BranchID: testBranchID, Code: testRegisterCode, IsActive: true}, nil).
		AnyTimes()
	store.EXPECT().
		TouchBranchRegister(gomock.Any(), gomock.Eq(testRegisterID)).
		Return(nil).
		AnyTimes()
}

func newBranchRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, path, reader)
	require.NoError(t, err)
	req.Header.Set(branchCodeHeaderKey, testBranchCode)
	req.Header.Set(branchKeyHeaderKey, testBranchKey)
	req.Header.Set("Content-Type", "application/json")
	return req
}
