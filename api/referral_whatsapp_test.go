package api

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReferralRewardText(t *testing.T) {
	ends := sql.NullTime{Time: time.Date(2026, 12, 20, 0, 0, 0, 0, time.Local), Valid: true}
	require.Equal(t,
		"Hello Rami! A friend just shopped at Centro with your referral code. You've earned 15% off your next purchase. Use it by 19 Dec 2026. Thank you for sharing!",
		referralRewardText("  Rami Haddad ", "Centro", 15, ends))
	require.Equal(t,
		"Hello! A friend just shopped at Hara with your referral code. You've earned 10% off your next purchase. Thank you for sharing!",
		referralRewardText("", "Hara", 10, sql.NullTime{}))
	require.NotRegexp(t, "[–—·•]", referralRewardText("Rami", "Centro", 15, ends))
}
