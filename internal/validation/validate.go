package validation

import (
"errors"
"fmt"
"html"
"strings"
)

// SanitizeInput trims and HTML-escapes user input to prevent XSS attacks.
func SanitizeInput(input string) string {
trimmed := strings.TrimSpace(input)
return html.EscapeString(trimmed)
}

// ValidateNonEmpty checks if a string is non-empty after trimming.
func ValidateNonEmpty(value, fieldName string) error {
if strings.TrimSpace(value) == "" {
return fmt.Errorf("%s is required", fieldName)
}
return nil
}

// ValidatePrices checks if prices are non-negative.
func ValidatePrices(lessonPrice, subscriptionPrice float64) error {
if lessonPrice < 0 || subscriptionPrice < 0 {
return errors.New("prices must be >= 0")
}
return nil
}

// ValidateDiscountPct checks if the discount percentage is within valid range.
func ValidateDiscountPct(discountPct float64) error {
if discountPct < 0 || discountPct > 100 {
return errors.New("discountPct must be between 0 and 100")
}
return nil
}
