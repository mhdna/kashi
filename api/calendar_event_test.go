package api

import (
	"encoding/json"
	"testing"

	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func TestNewSyncCalendarSettings(t *testing.T) {
	got := newSyncCalendarSettings(db.AppSetting{
		BusinessType:        "clothing",
		HiddenBuiltinEvents: json.RawMessage(`["Halloween"]`),
	})
	require.Equal(t, syncCalendarSettings{BusinessType: "clothing", HiddenEvents: []string{"Halloween"}}, got)

	// A broken or empty list shows everything rather than failing the pull.
	got = newSyncCalendarSettings(db.AppSetting{BusinessType: "tech", HiddenBuiltinEvents: json.RawMessage(`oops`)})
	require.Equal(t, []string{}, got.HiddenEvents)
}
