package field_format

import "testing"

func TestRegexpMapping(t *testing.T) {
	input := []struct {
		origin   string
		expected string
		rules    []RegexpMappingRule
	}{
		{
			origin:   "1234567890",
			expected: "numbers",
			rules: []RegexpMappingRule{
				{
					Regexp:  `\d+`,
					Mapping: `numbers`,
				},
			},
		},
		{
			origin:   "/api/v1/users/1234567890",
			expected: "/api/v1/users",
			rules: []RegexpMappingRule{
				{
					Regexp:  `^/api/v1/comments/(\d{10})$`,
					Mapping: `/api/v1/comments`,
				},
				{
					Regexp:  `^/api/v1/users/(\d{10})$`,
					Mapping: `/api/v1/users`,
				},
			},
		},
	}
	for _, test := range input {
		regexpMapping, err := NewRegexpMapping(RegexpMappingSettings{
			Rules: test.rules,
		})
		if err != nil {
			t.Errorf("NewRegexpMapping(%v) error: %v", test.rules, err)
			continue
		}
		result, err := regexpMapping.ParseString(test.origin)
		if err != nil {
			t.Errorf("ParseString(%s) error: %v", test.origin, err)
			continue
		}
		if result != test.expected {
			t.Errorf("ParseString(%s) expected %s, but got %s", test.origin, test.expected, result)
		}
	}
}
