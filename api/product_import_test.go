package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/lib/pq"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func uploadProductsCSV(t *testing.T, server *Server, content string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", "products.csv")
	require.NoError(t, err)
	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req, err := http.NewRequest(http.MethodPost, "/products/import", &body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", w.FormDataContentType())
	addAuthorization(t, req, server.tokenMaker, authorizationTypeBearer, "operator", time.Minute)

	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)
	return rec
}

func TestImportProductsCSV_CreatesAndSkipsExisting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)

	var got []db.CreateProductTxParams
	store.EXPECT().CreateProductTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, arg db.CreateProductTxParams) (db.CreateProductTxResult, error) {
			got = append(got, arg)
			if arg.Code == "BZDHMJ15" {
				return db.CreateProductTxResult{}, &pq.Error{Code: "23505"}
			}
			return db.CreateProductTxResult{}, nil
		}).Times(3)

	server := newTestServer(t, store)
	rec := uploadProductsCSV(t, server, "\uFEFFCode,Name,Price\n"+
		"BZDHMP600,Men Polo,6.70\n"+
		"BZDHMJ15,Men Jacket,$15\n"+
		",,\n"+
		"BZDHLSLSD200,\"Ladies Summer, Long Sleeve & Dresses\",2.23\n")

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var res struct {
		Created int      `json:"created"`
		Skipped []string `json:"skipped"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	require.Equal(t, 2, res.Created)
	require.Equal(t, []string{"BZDHMJ15"}, res.Skipped)

	require.Len(t, got, 3)
	require.Equal(t, int64(670), *got[0].Price)
	require.Equal(t, int64(1500), *got[1].Price)
	require.Equal(t, "Ladies Summer, Long Sleeve & Dresses", got[2].Name)
	require.Nil(t, got[0].PriceListID)
}

func TestImportProductsCSV_BadRowsImportNothing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().CreateProductTx(gomock.Any(), gomock.Any()).Times(0)

	server := newTestServer(t, store)
	rec := uploadProductsCSV(t, server, "code,name,price\n"+
		"BZDHMP600,Men Polo,6.70\n"+
		"BZDHMP600,Men Polo again,6.70\n"+
		"BZDHX,,abc\n")

	require.Equal(t, http.StatusBadRequest, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "row 3: code BZDHMP600 already used on row 2")
	require.Contains(t, body, "row 4: missing name")
	require.Contains(t, body, `row 4: price \"abc\" is not a dollar amount`)
}

func TestImportProductsCSV_RequiresColumns(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().CreateProductTx(gomock.Any(), gomock.Any()).Times(0)

	server := newTestServer(t, store)
	rec := uploadProductsCSV(t, server, "code,name\nBZDHMP600,Men Polo\n")
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
