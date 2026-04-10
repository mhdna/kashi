package util

func CalculateNetAmount(amount int64, discount int16) (int64, error) {
	netAmount := amount * (100 - int64(discount)) / 100
	return netAmount, nil
}
