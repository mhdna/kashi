package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCapitalizeWords(t *testing.T) {
	for in, want := range map[string]string{
		"alex":           "Alex",
		"  alex   sal  ": "Alex Sal",
		"McDonald":        "McDonald",
		"élie":            "Élie",
		"محمد علي":        "محمد علي",
		"":                "",
	} {
		require.Equal(t, want, CapitalizeWords(in), in)
	}
}
