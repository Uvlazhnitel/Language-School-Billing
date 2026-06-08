package money

import "math"

func EurosToCents(amount float64) int64 {
	return int64(math.Round(amount * 100))
}

func CentsToEuros(cents int64) float64 {
	return float64(cents) / 100
}

func MulFloatToCents(qty float64, unitCents int64) int64 {
	return int64(math.Round(qty * float64(unitCents)))
}
