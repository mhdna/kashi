package api

import (
	"bytes"
	"database/sql"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// newShiftReceiptImageRequest builds the multipart POST kashi-pos's uploader
// sends: a shift_ref field plus one image part with the given content type.
func newShiftReceiptImageRequest(t *testing.T, shiftRef, imageContentType string, imageBytes []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	require.NoError(t, mw.WriteField("shift_ref", shiftRef))

	header := make(map[string][]string)
	header["Content-Disposition"] = []string{`form-data; name="image"; filename="receipt.jpg"`}
	if imageContentType != "" {
		header["Content-Type"] = []string{imageContentType}
	}
	part, err := mw.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write(imageBytes)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req, err := http.NewRequest(http.MethodPost, "/branch/shift_receipt_images", &body)
	require.NoError(t, err)
	req.Header.Set(branchCodeHeaderKey, testBranchCode)
	req.Header.Set(branchKeyHeaderKey, testBranchKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestCreateBranchShiftReceiptImage_ConflictBeforeShiftClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	store.EXPECT().
		GetBranchShiftByClientRef(gomock.Any(), gomock.Eq(db.GetBranchShiftByClientRefParams{BranchID: testBranchID, ClientRef: "shift-9"})).
		Return(db.BranchShift{}, sql.ErrNoRows)

	server := newTestServer(t, store)
	server.config.ShiftReceiptImageStorageDir = t.TempDir()
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, newShiftReceiptImageRequest(t, "shift-9", "image/jpeg", []byte("data")))

	require.Equal(t, http.StatusConflict, recorder.Code)
}

func TestCreateBranchShiftReceiptImage_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	store.EXPECT().
		GetBranchShiftByClientRef(gomock.Any(), gomock.Any()).
		Return(db.BranchShift{ID: 55, BranchID: testBranchID, ClientRef: "shift-9", ClosingPersonName: "Dana"}, nil)
	store.EXPECT().
		GetBranchShiftReceiptImageForShift(gomock.Any(), gomock.Eq(int64(55))).
		Return(db.BranchShiftReceiptImage{}, sql.ErrNoRows)
	store.EXPECT().
		CreateBranchShiftReceiptImage(gomock.Any(), gomock.Any()).
		Times(1).
		Return(db.BranchShiftReceiptImage{ID: 1, BranchShiftID: 55}, nil)

	server := newTestServer(t, store)
	server.config.ShiftReceiptImageStorageDir = t.TempDir()
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, newShiftReceiptImageRequest(t, "shift-9", "image/jpeg", []byte("JPEGDATA")))

	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestCreateBranchShiftReceiptImage_IdempotentWhenAlreadyUploaded(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	store.EXPECT().
		GetBranchShiftByClientRef(gomock.Any(), gomock.Any()).
		Return(db.BranchShift{ID: 55, BranchID: testBranchID, ClientRef: "shift-9"}, nil)
	store.EXPECT().
		GetBranchShiftReceiptImageForShift(gomock.Any(), gomock.Eq(int64(55))).
		Return(db.BranchShiftReceiptImage{ID: 1, BranchShiftID: 55}, nil)
	// A second upload must not create another row.
	store.EXPECT().CreateBranchShiftReceiptImage(gomock.Any(), gomock.Any()).Times(0)

	server := newTestServer(t, store)
	server.config.ShiftReceiptImageStorageDir = t.TempDir()
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, newShiftReceiptImageRequest(t, "shift-9", "image/jpeg", []byte("JPEGDATA")))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "already_uploaded")
}

func TestCreateBranchShiftReceiptImage_RejectsUnsupportedType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	store.EXPECT().
		GetBranchShiftByClientRef(gomock.Any(), gomock.Any()).
		Return(db.BranchShift{ID: 55, BranchID: testBranchID, ClientRef: "shift-9"}, nil)
	store.EXPECT().
		GetBranchShiftReceiptImageForShift(gomock.Any(), gomock.Eq(int64(55))).
		Return(db.BranchShiftReceiptImage{}, sql.ErrNoRows)
	store.EXPECT().CreateBranchShiftReceiptImage(gomock.Any(), gomock.Any()).Times(0)

	server := newTestServer(t, store)
	server.config.ShiftReceiptImageStorageDir = t.TempDir()
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, newShiftReceiptImageRequest(t, "shift-9", "application/pdf", []byte("%PDF-1.4")))

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}
