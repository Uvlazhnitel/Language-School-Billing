package validation

import (
	"testing"
)

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal text",
			input:    "John Doe",
			expected: "John Doe",
		},
		{
			name:     "text with spaces",
			input:    "  John Doe  ",
			expected: "John Doe",
		},
		{
			name:     "HTML tags",
			input:    "<script>alert('xss')</script>",
			expected: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			name:     "special characters",
			input:    "Test & <test>",
			expected: "Test &amp; &lt;test&gt;",
		},
		{
			name:     "quotes",
			input:    `He said "Hello"`,
			expected: "He said &#34;Hello&#34;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeInput(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateNonEmpty(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		fieldName string
		wantError bool
	}{
		{"valid value", "test", "field", false},
		{"empty string", "", "field", true},
		{"only spaces", "   ", "field", true},
		{"with spaces", " test ", "field", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNonEmpty(tt.value, tt.fieldName)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateNonEmpty() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidatePrices(t *testing.T) {
	tests := []struct {
		name              string
		lessonPrice       float64
		subscriptionPrice float64
		wantError         bool
	}{
		{"valid prices", 10.0, 100.0, false},
		{"zero prices", 0.0, 0.0, false},
		{"negative lesson price", -10.0, 100.0, true},
		{"negative subscription price", 10.0, -100.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePrices(tt.lessonPrice, tt.subscriptionPrice)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidatePrices() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateDiscountPct(t *testing.T) {
	tests := []struct {
		name        string
		discountPct float64
		wantError   bool
	}{
		{"valid 0%", 0.0, false},
		{"valid 50%", 50.0, false},
		{"valid 100%", 100.0, false},
		{"invalid negative", -10.0, true},
		{"invalid over 100", 150.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDiscountPct(tt.discountPct)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateDiscountPct() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
