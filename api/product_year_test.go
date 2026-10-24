package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithDefaultYear(t *testing.T) {
	added := time.Date(2026, time.October, 23, 12, 0, 0, 0, time.UTC)

	require.Equal(t, "2026", withDefaultYear(Attributes{}, added).Year)
	require.Equal(t, "2026", withDefaultYear(Attributes{Year: "  "}, added).Year)
	require.Equal(t, "2024", withDefaultYear(Attributes{Year: "2024"}, added).Year)
}
