package api

import (
	"database/sql"
	"math/rand"
	"net/http"
	"testing"
	"time"

	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func TestOpenInvoiceExchangeQueuesForTheSalesTill(t *testing.T) {
	for name, until := range map[string]string{
		"only today":  time.Now().Format("2006-01-02"),
		"for 10 days": time.Now().AddDate(0, 0, 9).Format("2006-01-02"),
	} {
		t.Run(name, func(t *testing.T) {
			code, body, sent := queueCommand(t, "open_invoice_exchange", map[string]any{
				"client_ref": "sale-1", "until_date": until, "reason": "customer travelling",
			}, sale)
			require.Equal(t, http.StatusOK, code, body)
			require.JSONEq(t, `"sale-1"`, string(sent["client_ref"]))
			require.JSONEq(t, `"`+until+`"`, string(sent["until_date"]))
			require.JSONEq(t, `"customer travelling"`, string(sent["reason"]))
			require.NotEmpty(t, string(sent["opened_by"]))
		})
	}
}

// An admin can't name who opened it: kashi stamps the signed-in user.
func TestOpenInvoiceExchangeStampsTheAdmin(t *testing.T) {
	_, _, sent := queueCommand(t, "open_invoice_exchange", map[string]any{
		"client_ref": "sale-1", "until_date": time.Now().Format("2006-01-02"), "opened_by": "someone else",
	}, sale)
	require.NotEqual(t, `"someone else"`, string(sent["opened_by"]))
}

func TestOpenInvoiceExchangeRefusals(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	cancelled := sale
	cancelled.CancelledAt = sql.NullTime{Valid: true}
	imported := sale
	imported.Historical = true
	ret := sale
	ret.Kind = "return"
	exchange := sale
	exchange.Kind = "exchange"
	cases := map[string]struct {
		payload map[string]any
		invoice db.BranchInvoice
		want    string
	}{
		"no client_ref":  {map[string]any{"until_date": today}, sale, "client_ref"},
		"no date":        {map[string]any{"client_ref": "sale-1"}, sale, "until_date"},
		"bad date":       {map[string]any{"client_ref": "sale-1", "until_date": "25/11/2026"}, sale, "until_date"},
		"past date":      {map[string]any{"client_ref": "sale-1", "until_date": time.Now().AddDate(0, 0, -3).Format("2006-01-02")}, sale, "already passed"},
		"too far":        {map[string]any{"client_ref": "sale-1", "until_date": time.Now().AddDate(2, 0, 0).Format("2006-01-02")}, sale, "a year"},
		"cancelled sale": {map[string]any{"client_ref": "sale-1", "until_date": today}, cancelled, "cancelled"},
		"imported sale":  {map[string]any{"client_ref": "sale-1", "until_date": today}, imported, "old POS"},
		"a return":       {map[string]any{"client_ref": "sale-1", "until_date": today}, ret, "only a sale"},
		"an exchange":    {map[string]any{"client_ref": "sale-1", "until_date": today}, exchange, "only a sale"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			code, body, sent := queueCommand(t, "open_invoice_exchange", c.payload, c.invoice)
			require.Equal(t, http.StatusBadRequest, code, body)
			require.Contains(t, body, c.want)
			require.Nil(t, sent, "nothing is queued")
		})
	}
}

// Any date from yesterday to a year ahead passes, anything else is refused,
// whatever the time of day.
func TestCheckExchangeOpeningDateRandomized(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	for i := 0; i < 500; i++ {
		now := time.Date(2026, time.Month(1+rng.Intn(12)), 1+rng.Intn(28), rng.Intn(24), rng.Intn(60), 0, 0, time.UTC)
		offset := rng.Intn(800) - 400
		err := checkExchangeOpeningDate(now.AddDate(0, 0, offset).Format("2006-01-02"), now)
		if offset >= -1 && offset <= maxExchangeOpeningDays {
			require.NoError(t, err, "offset %d at %v", offset, now)
		} else {
			require.Error(t, err, "offset %d at %v", offset, now)
		}
	}
}
