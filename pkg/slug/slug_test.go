package slug

import (
	"testing"
)

func TestGenerate(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Standard Station",
			input:    "Bundaran HI",
			expected: "bundaran-hi",
		},
		{
			name:     "Station with Commercial Partner",
			input:    "Dukuh Atas BNI",
			expected: "dukuh-atas-bni",
		},
		{
			name:     "Station with Slash Delimiter",
			input:    "CSW / ASEAN",
			expected: "csw-asean",
		},
		{
			name:     "Station with Periods and Slash",
			input:    "M.H. Thamrin / Sarinah",
			expected: "mh-thamrin-sarinah",
		},
		{
			name:     "Station with Slash and Bank",
			input:    "Kebon Sirih / Bank Indonesia",
			expected: "kebon-sirih-bank-indonesia",
		},
		{
			name:     "Mikrotrans Route with Hyphens and Via",
			input:    "Tanah Abang - Grogol via Tawakal",
			expected: "tanah-abang-grogol-via-tawakal",
		},
		{
			name:     "Route with Parentheses and Hyphens",
			input:    "MRT North-South Line (Lebak Bulus - Bundaran HI)",
			expected: "mrt-north-south-line-lebak-bulus-bundaran-hi",
		},
		{
			name:     "Corridor Route",
			input:    "Koridor 1 (Blok M - Kota)",
			expected: "koridor-1-blok-m-kota",
		},
		{
			name:     "Empty String",
			input:    "",
			expected: "",
		},
		{
			name:     "Whitespace Only",
			input:    "   ",
			expected: "",
		},
		{
			name:     "Multiple Special Characters",
			input:    "---Stasiun + Kota & Tua---",
			expected: "stasiun-kota-tua",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Generate(tc.input)
			if result != tc.expected {
				t.Errorf("Generate(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
			resultAlias := GenerateSlug(tc.input)
			if resultAlias != tc.expected {
				t.Errorf("GenerateSlug(%q) = %q, expected %q", tc.input, resultAlias, tc.expected)
			}
		})
	}
}
