package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// /app_settings is readable by every signed-in user, so it never carries the
// WhatsApp access token (that's /whatsapp_settings' job).
func TestGetAppSettings_HidesWhatsAppToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetAppSettings(gomock.Any()).
		Return(db.AppSetting{ID: 1, WhatsappAccessToken: "secret-token"}, nil)

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodGet, "/app_settings", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotContains(t, rec.Body.String(), "secret-token")
}

func TestSetProductAutofillRules_SavesRulesInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().SetProductAutofillRules(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.SetProductAutofillRulesParams) (db.AppSetting, error) {
			var saved []productAutofillRule
			require.NoError(t, json.Unmarshal(arg.ProductAutofillRules, &saved))
			require.Len(t, saved, 3)
			require.Equal(t, "origin", saved[0].Field)
			require.Equal(t, "price", saved[1].Field)
			require.Equal(t, "type", saved[2].Part)
			require.Equal(t, `^(?<origin>[A-Z]+)(?<type>[A-Z]+)$`, arg.ProductAutofillCodeLayout)
			return db.AppSetting{ID: 1, ProductAutofillRules: arg.ProductAutofillRules, WhatsappAccessToken: "secret-token"}, nil
		})

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPut, "/app_settings/product_autofill_rules", map[string]any{
		"rules": []map[string]any{
			{"source": "code", "match": "char", "position": 1, "pattern": "L", "field": "origin", "value": "Lebanon"},
			{"source": "code", "match": "trailing_number", "pattern": "", "field": "price", "value": ""},
			{"source": "code", "match": "part", "part": "type", "pattern": "JE", "field": "type", "value": "Jeans"},
		},
		"code_layout": `^(?<origin>[A-Z]+)(?<type>[A-Z]+)$`,
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotContains(t, rec.Body.String(), "secret-token")
}

func TestSetProductAutofillRules_RejectsBadRules(t *testing.T) {
	cases := map[string]map[string]any{
		"unknown source": {"source": "barcode", "match": "prefix", "pattern": "L", "field": "origin", "value": "Lebanon"},
		"unknown match":  {"source": "code", "match": "fuzzy", "pattern": "L", "field": "origin", "value": "Lebanon"},
		"unknown field":  {"source": "code", "match": "prefix", "pattern": "L", "field": "colour", "value": "Red"},
		"char at 0":      {"source": "code", "match": "char", "position": 0, "pattern": "L", "field": "origin", "value": "Lebanon"},
		"empty pattern":  {"source": "name", "match": "word", "pattern": "", "field": "type", "value": "Shirt"},
		"part, no part":  {"source": "code", "match": "part", "pattern": "JE", "field": "type", "value": "Jeans"},
		"part of name":   {"source": "name", "match": "part", "part": "type", "pattern": "JE", "field": "type", "value": "Jeans"},
	}
	for name, rule := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl) // no calls expected: nothing is saved

			server := newTestServer(t, store)
			rec := adminRequest(t, server, http.MethodPut, "/app_settings/product_autofill_rules", map[string]any{
				"rules": []map[string]any{rule},
			})
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		})
	}
}

func validAppSettingsBody(businessType string) map[string]any {
	return map[string]any{
		"financials_high_season_months": []int{6, 7},
		"activity_message_seconds":      5,
		"activity_display_mode":         "appbar",
		"barcode_label_width":           50,
		"barcode_label_height":          30,
		"upcoming_events_days":          15,
		"upcoming_events_menu_days":     180,
		"table_page_size":               20,
		"business_type":                 businessType,
	}
}

func TestUpdateAppSettings_BusinessType(t *testing.T) {
	t.Run("unknown type is refused", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl) // nothing saved
		server := newTestServer(t, store)
		rec := adminRequest(t, server, http.MethodPut, "/app_settings", validAppSettingsBody("bakery"))
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	})

	t.Run("a missing type saves as clothing", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		store.EXPECT().UpdateAppSettings(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, arg db.UpdateAppSettingsParams) (db.AppSetting, error) {
				require.Equal(t, "clothing", arg.BusinessType)
				return db.AppSetting{ID: 1, BusinessType: arg.BusinessType}, nil
			})
		server := newTestServer(t, store)
		rec := adminRequest(t, server, http.MethodPut, "/app_settings", validAppSettingsBody(""))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	})

	t.Run("tech is saved", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		store.EXPECT().UpdateAppSettings(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, arg db.UpdateAppSettingsParams) (db.AppSetting, error) {
				require.Equal(t, "tech", arg.BusinessType)
				return db.AppSetting{ID: 1, BusinessType: arg.BusinessType}, nil
			})
		server := newTestServer(t, store)
		rec := adminRequest(t, server, http.MethodPut, "/app_settings", validAppSettingsBody("tech"))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	})
}
