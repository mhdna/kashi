package api

import (
	"database/sql"
	"encoding/json"
	"testing"

	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// The admin pages read image_id as a number and waived_at as a time (or
// null), not as the {"Int64":..,"Valid":..} a raw sql.Null* marshals to.
func TestUnknownItemViewsFlattenNullables(t *testing.T) {
	raw, err := json.Marshal(unknownItemViews([]db.ListUnknownItemsRow{
		{ID: 1, ImageID: sql.NullInt64{Int64: 7, Valid: true}},
		{ID: 2},
	}))
	require.NoError(t, err)
	var out []map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))
	require.EqualValues(t, 7, out[0]["image_id"])
	require.Nil(t, out[1]["image_id"])
	require.Nil(t, out[1]["waived_at"])

	raw, err = json.Marshal(waitingUnknownItemViews([]db.ListWaitingUnknownItemsRow{{ID: 3, ImageID: sql.NullInt64{Int64: 9, Valid: true}}}))
	require.NoError(t, err)
	require.Contains(t, string(raw), `"image_id":9`)
}
