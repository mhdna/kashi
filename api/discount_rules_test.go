package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func putRules(t *testing.T, store *mockdb.MockStore, body any) *httptest.ResponseRecorder {
	t.Helper()
	server := newTestServer(t, store)
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPut, "/discount_lists/1/rules", bytes.NewReader(raw))
	require.NoError(t, err)
	addAuthorization(t, req, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)
	return rec
}

// A page from before groups sends "rules"; saving that as no groups would
// delete every rule, so nothing is written.
func TestSetDiscountListRulesRefusesAStalePage(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ReplaceDiscountListRuleGroupsTx(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	rec := putRules(t, store, map[string]any{"rules": []any{}})
	require.Equal(t, http.StatusConflict, rec.Code)
}

// Groups and their rules are saved in order with their switches.
func TestSetDiscountListRuleGroups(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetDiscountList(gomock.Any(), int64(1)).Return(db.DiscountList{ID: 1}, nil)
	store.EXPECT().ReplaceDiscountListRuleGroupsTx(gomock.Any(), int64(1), gomock.Any()).
		DoAndReturn(func(_ any, _ int64, groups []db.DiscountListRuleGroupInput) error {
			require.Len(t, groups, 2)
			require.Equal(t, "Exclusions", groups[0].Name)
			require.True(t, groups[0].Enabled)
			require.Equal(t, []string{"Accessories"}, groups[0].Rules[0].Rule.Categories)
			require.False(t, groups[1].Enabled)
			require.False(t, groups[1].Rules[0].Rule.IsEnabled)
			return nil
		})
	store.EXPECT().ListDiscountListRuleGroups(gomock.Any(), int64(1)).Return([]db.DiscountListRuleGroup{}, nil)
	store.EXPECT().ListDiscountListRules(gomock.Any(), int64(1)).Return([]db.ListDiscountListRulesRow{}, nil)
	store.EXPECT().ListDiscountListRuleOptions(gomock.Any(), int64(1)).Return([]db.ListDiscountListRuleOptionsRow{}, nil)

	rec := putRules(t, store, map[string]any{"groups": []any{
		map[string]any{"name": "Exclusions", "enabled": true, "rules": []any{
			map[string]any{"name": "No accessories", "categories": []string{"Accessories"}, "enabled": true},
		}},
		map[string]any{"name": "Winter", "enabled": false, "rules": []any{
			map[string]any{"name": "CA", "code_prefixes": []string{"CA"}, "discount": 20, "enabled": false},
		}},
	}})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}
