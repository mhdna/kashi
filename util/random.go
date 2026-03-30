package util

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func init() {
	rand.Seed(time.Now().UnixNano())
}

func RandomInt(min, max int64) int64 {
	return min + rand.Int63n(max-min+1)
}

func RandomMoneyAmount() string {
	v := int64(1) + rand.Int63n(100000)
	return fmt.Sprintf("%d.0000", v)
}

func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[rand.Intn(k)]
		sb.WriteByte(c)
	}
	return sb.String()
}

func RandomCode() string {
	return RandomString(20)
}

func RandomPhone() string {
	// US style phone number
	n := rand.Int63n(1_000_000_0000)
	return fmt.Sprintf("%010d", n)
}

func RandomName() string {
	return RandomString(10)
}

func RandomAttributeValue() string {
	return RandomString(6)
}

func RandomNumber() int64 {
	return RandomInt(1, 100)
}

func RandomQuantity() int64 {
	return RandomInt(1, 1000)
}

func RandomLongitudeLatitude() sql.NullFloat64 {
	return sql.NullFloat64{
		Float64: float64(RandomInt(1, 100)),
		Valid:   true,
	}
}

func RandomCurrency() string {
	currencies := []string{"USD", "LBP", "EUR"}
	n := len(currencies)
	return currencies[rand.Intn(n)]
}
