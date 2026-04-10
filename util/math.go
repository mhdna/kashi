package util

import "strconv"

func CalculateNetAmount(amount string, discount int16) (string, error) {
	amt, err := strconv.ParseFloat(amount, 10)
	if err != nil {
		return "", err
	}
	d := int16(0)
	netAmount := amt * (100 - float64(d)) / 100
	return strconv.FormatFloat(netAmount, 'f', 0, 64), nil
}

func NegateAmount(amount string) (string, error) {
	amt, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(-amt, 10), nil
}
