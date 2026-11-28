package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// The options a till seeds offline have these uids (kashi-pos migrations
// 000151 to 000153): kashi must hold the same ones so a till's first answers
// land.
var seededClientSources = map[string]string{
	"6f1c2a00-5e1d-4c7a-9b00-000000000001": "Social media",
	"6f1c2a00-5e1d-4c7a-9b00-000000000003": "A friend or family",
	"6f1c2a00-5e1d-4c7a-9b00-000000000004": "Walking by",
}

// SMS and Other (migration 000261), then Google Maps, WhatsApp and An ad or
// flyer (000262) were dropped; a till never offers them.
var droppedClientSources = []string{
	"6f1c2a00-5e1d-4c7a-9b00-000000000002",
	"6f1c2a00-5e1d-4c7a-9b00-000000000005",
	"6f1c2a00-5e1d-4c7a-9b00-000000000006",
	"6f1c2a00-5e1d-4c7a-9b00-000000000007",
	"6f1c2a00-5e1d-4c7a-9b00-000000000008",
}

func TestClientSourcesSeededWithTillUIDs(t *testing.T) {
	rows, err := testQueries.ListClientSourcesForSync(context.Background())
	require.NoError(t, err)
	found := map[string]ClientSource{}
	for _, r := range rows {
		found[r.Uid.String()] = r
	}
	for uid, name := range seededClientSources {
		require.Equal(t, name, found[uid].Name, "seeded option %s", uid)
		require.True(t, found[uid].IsActive, "seeded option %s", uid)
	}
	for _, uid := range droppedClientSources {
		if r, ok := found[uid]; ok {
			require.False(t, r.IsActive, "dropped option %s still offered", uid)
		}
	}
}

func TestSetClientSourceFromTill(t *testing.T) {
	ctx := context.Background()
	friend, err := testQueries.CreateClientSource(ctx, CreateClientSourceParams{Name: "Test friend " + uuid.NewString(), SortOrder: 90, IsActive: true})
	require.NoError(t, err)
	sms, err := testQueries.CreateClientSource(ctx, CreateClientSourceParams{Name: "Test sms " + uuid.NewString(), SortOrder: 91, IsActive: true})
	require.NoError(t, err)

	t.Run("a new client takes the till's answer", func(t *testing.T) {
		c := createRandomClient(t)
		n, err := testQueries.SetClientSourceFromTill(ctx, SetClientSourceFromTillParams{SourceUid: friend.Uid, ClientID: c.ID})
		require.NoError(t, err)
		require.EqualValues(t, 1, n)
		got, err := testQueries.GetClient(ctx, c.ID)
		require.NoError(t, err)
		require.Equal(t, sql.NullInt64{Int64: friend.ID, Valid: true}, got.SourceID)
		// updated_at stays: tills don't pull a client's source.
		require.True(t, got.UpdatedAt.Equal(c.UpdatedAt))
	})

	t.Run("the first answer wins", func(t *testing.T) {
		c := createRandomClient(t)
		_, err := testQueries.SetClientSourceFromTill(ctx, SetClientSourceFromTillParams{SourceUid: friend.Uid, ClientID: c.ID})
		require.NoError(t, err)
		n, err := testQueries.SetClientSourceFromTill(ctx, SetClientSourceFromTillParams{SourceUid: sms.Uid, ClientID: c.ID})
		require.NoError(t, err)
		require.EqualValues(t, 0, n)
		got, err := testQueries.GetClient(ctx, c.ID)
		require.NoError(t, err)
		require.Equal(t, friend.ID, got.SourceID.Int64)
	})

	t.Run("an unknown option leaves the client alone", func(t *testing.T) {
		c := createRandomClient(t)
		n, err := testQueries.SetClientSourceFromTill(ctx, SetClientSourceFromTillParams{SourceUid: uuid.New(), ClientID: c.ID})
		require.NoError(t, err)
		require.EqualValues(t, 0, n)
		got, err := testQueries.GetClient(ctx, c.ID)
		require.NoError(t, err)
		require.False(t, got.SourceID.Valid)
	})

	t.Run("the list counts each option's clients", func(t *testing.T) {
		before := map[int64]int64{}
		rows, err := testQueries.ListClientSources(ctx)
		require.NoError(t, err)
		for _, r := range rows {
			before[r.ID] = r.ClientCount
		}
		for i := 0; i < 3; i++ {
			c := createRandomClient(t)
			_, err := testQueries.SetClientSourceFromTill(ctx, SetClientSourceFromTillParams{SourceUid: sms.Uid, ClientID: c.ID})
			require.NoError(t, err)
		}
		rows, err = testQueries.ListClientSources(ctx)
		require.NoError(t, err)
		for _, r := range rows {
			if r.ID == sms.ID {
				require.Equal(t, before[sms.ID]+3, r.ClientCount)
			}
		}
		n, err := testQueries.CountClientsWithSource(ctx, sql.NullInt64{Int64: sms.ID, Valid: true})
		require.NoError(t, err)
		require.Equal(t, before[sms.ID]+3, n)
	})

	t.Run("an option clients gave can't be deleted", func(t *testing.T) {
		require.Error(t, testQueries.DeleteClientSource(ctx, sms.ID))
		unused, err := testQueries.CreateClientSource(ctx, CreateClientSourceParams{Name: "Unused " + uuid.NewString(), IsActive: true})
		require.NoError(t, err)
		require.NoError(t, testQueries.DeleteClientSource(ctx, unused.ID))
	})
}
