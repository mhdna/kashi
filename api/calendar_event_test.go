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
	require.Equal(t, syncCalendarSettings{BusinessType: "clothing", HiddenEvents: []string{"Halloween"}, Catalog: builtinEventsCatalog}, got)

	// A broken or empty list shows everything rather than failing the pull.
	got = newSyncCalendarSettings(db.AppSetting{BusinessType: "tech", HiddenBuiltinEvents: json.RawMessage(`oops`)})
	require.Equal(t, []string{}, got.HiddenEvents)
}

// The catalog is what the tills and kashi's own calendar both read: it must
// parse, every event must use a known list and rule, and the business types
// must be the ones the settings API accepts.
func TestBuiltinEventsCatalog(t *testing.T) {
	var catalog struct {
		Icons         []string                   `json:"icons"`
		Lists         map[string]json.RawMessage `json:"lists"`
		BusinessTypes map[string]struct {
			Lists []string `json:"lists"`
		} `json:"businessTypes"`
		Events []struct {
			List string `json:"list"`
			Name string `json:"name"`
			Icon string `json:"icon"`
			Rule struct {
				Type string `json:"type"`
			} `json:"rule"`
		} `json:"events"`
	}
	require.NoError(t, json.Unmarshal(builtinEventsCatalog, &catalog))

	require.Len(t, catalog.BusinessTypes, len(validBusinessTypes))
	for key, bt := range catalog.BusinessTypes {
		require.True(t, validBusinessTypes[key], "business type %q is not accepted by the settings API", key)
		for _, list := range bt.Lists {
			require.Contains(t, catalog.Lists, list)
		}
	}

	// Every icon the catalog itself uses is in the shared set, which the tills
	// must all have (their build checks the same file).
	for _, e := range catalog.Events {
		require.True(t, calendarIconAllowed(e.Icon), "%s uses %s, which is not in icons", e.Name, e.Icon)
	}
	require.True(t, calendarIconAllowed("mdi-calendar-star"), "the default icon must be allowed")
	require.False(t, calendarIconAllowed("mdi-not-a-real-icon"))

	rules := map[string]bool{"fixed": true, "span": true, "nthWeekday": true, "easter": true, "hijri": true}
	seen := map[string]bool{}
	for _, e := range catalog.Events {
		require.Contains(t, catalog.Lists, e.List, e.Name)
		require.True(t, rules[e.Rule.Type], "%s: unknown rule %q", e.Name, e.Rule.Type)
		require.False(t, seen[e.Name], "duplicate event name %q (names are how events are hidden)", e.Name)
		seen[e.Name] = true
	}
}
