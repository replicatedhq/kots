package snapshot

import "testing"

func TestFormatTTL(t *testing.T) {
	tests := []struct {
		quantity string
		unit     string
		duration string
	}{
		{"1000", "seconds", "1000s"},
		{"500", "minutes", "500m"},
		{"3", "years", "26298h"},
		{"5", "weeks", "840h"},
		{"2", "weeks", "336h"},
		{"6", "months", "4320h"},
		{"1", "days", "24h"},
	}
	for _, test := range tests {
		t.Run(test.duration, func(t *testing.T) {
			formatted, err := FormatTTL(test.quantity, test.unit)
			if err != nil {
				t.Fatal(err)
			}
			if formatted != test.duration {
				t.Errorf("Expected %q, got %q", test.duration, formatted)
			}
		})
	}

	if _, err := FormatTTL("three", "decades"); err == nil {
		t.Error("Expected error")
	}
}

func TestParseTTL(t *testing.T) {
	tests := []struct {
		quantity int64
		unit     string
		duration string
	}{
		{1000, "seconds", "1000s"},
		{500, "minutes", "500m"},
		{3, "years", "26298h"},
		{5, "weeks", "840h"},
		{2, "weeks", "336h"},
		{6, "months", "4320h"},
		{1, "days", "24h"},
	}
	for _, test := range tests {
		t.Run(test.duration, func(t *testing.T) {
			parsed, err := ParseTTL(test.duration)
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Quantity != test.quantity {
				t.Errorf("Expected quantity %d, got %d", test.quantity, parsed.Quantity)
			}
			if parsed.Unit != test.unit {
				t.Errorf("Expected unit %s, got %s", test.unit, parsed.Unit)
			}
		})
	}

	if parsed, err := ParseTTL("7years"); err == nil {
		t.Errorf("Expected error, got %v", parsed)
	}
}
