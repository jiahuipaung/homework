package field_format

import (
	"testing"
)

func TestDesensitize(t *testing.T) {
	input := []struct {
		method   string
		origin   string
		expected string
		regexp   string
		replace  string
	}{
		{
			method:   DesensitizeMethodPhone,
			origin:   "13812345678",
			expected: "138****5678",
		},
		{
			method:   DesensitizeMethodBankCard,
			origin:   "491648411333978312",
			expected: "****978312",
		},
		{
			method:   DesensitizeMethodEmail,
			origin:   "test@example.com",
			expected: "****@example.com",
		},
		{
			method:   DesensitizeMethodIP,
			origin:   "192.168.1.1",
			expected: "192.****",
		},
		{
			method:   DesensitizeMethodIDCard,
			origin:   "11010519491231001X",
			expected: "1101****",
		},
		{
			method:   DesensitizeMethodAccessKey,
			origin:   "rDhc9qxjhIhlBiyphP7buo5yg5h6Eq",
			expected: "rDhc****",
		},
		{
			method:   DesensitizeMethodKeepFirstLast,
			origin:   "1234567890",
			expected: "1****0",
		},
		{
			method:   DesensitizeMethodKeepLastFour,
			origin:   "1234567890",
			expected: "****7890",
		},
		{
			method:   DesensitizeMethodMd5,
			origin:   "1234567890",
			expected: "e807f1fcf82d132f9bb018ca6738a19f",
		},
		{
			method:   DesensitizeMethodCustom,
			origin:   "1234567890",
			expected: "1234****7890",
			regexp:   `(\d{4})\d{2}(\d{4})`,
			replace:  `$1****$2`,
		},
	}
	for _, test := range input {
		desensitize, err := NewDesensitize(DesensitizeSettings{
			Method:  test.method,
			Regexp:  test.regexp,
			Replace: test.replace,
		})
		if err != nil {
			t.Errorf("NewDesensitize(%s) error: %v", test.method, err)
			continue
		}
		result, err := desensitize.ParseString(test.origin)
		if err != nil {
			t.Errorf("ParseString(%s) error: %v", test.origin, err)
			continue
		}
		if result != test.expected {
			t.Errorf("ParseString(%s) expected %s, but got %s", test.origin, test.expected, result)
		}
	}
}
