package utils

import "math"

// Round2 rounds a float64 value to 2 decimal places.
// This is used throughout the application for currency calculations
// to ensure consistent rounding behavior.
//
// Example:
//   Round2(10.456) // returns 10.46
//   Round2(10.454) // returns 10.45
func Round2(v float64) float64 {
return math.Round(v*100) / 100
}
