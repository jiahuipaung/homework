package field_format

import (
	"testing"
	"time"

	"github.com/flashcatcloud/fc-stash/types"
)

func TestFieldFormat(t *testing.T) {
	tests := []struct {
		name         string
		settings     FieldFormatSettings
		value        interface{}
		expected     interface{}
		expectedType string
	}{
		{
			name: "test_1",
			settings: FieldFormatSettings{
				FormatType: PresetAllmatch,
			},
			value:        "123",
			expected:     "123",
			expectedType: types.LogExtractValueTypeText,
		},
		{
			name: "test_2",
			settings: FieldFormatSettings{
				FormatType: PresetAllmatch,
			},
			value:        "123.456",
			expected:     123.456,
			expectedType: types.LogExtractValueTypeFloat,
		},
		{
			name: "test_3",
			settings: FieldFormatSettings{
				FormatType: PresetAllmatch,
			},
			value:        123.456,
			expected:     "123.456",
			expectedType: types.LogExtractValueTypeText,
		},
		{
			name: "test_4",
			settings: FieldFormatSettings{
				FormatType: PresetRegexp,
				Regexp:     `\/(\d+)`,
			},
			value:        "/api/v1/users/123",
			expected:     int64(123),
			expectedType: types.LogExtractValueTypeLong,
		},
		{
			name: "test_5",
			settings: FieldFormatSettings{
				FormatType: PresetUriPath,
			},
			value:        "/api/v1/users/123",
			expected:     "/api/v1/users/:id",
			expectedType: types.LogExtractValueTypeText,
		},
		{
			name: "test_6",
			settings: FieldFormatSettings{
				FormatType: PresetDesensitize,
				Desensitize: DesensitizeSettings{
					Method: DesensitizeMethodIP,
				},
			},
			value:        "192.168.1.1",
			expected:     "192.****",
			expectedType: types.LogExtractValueTypeText,
		},
		{
			name: "test_7",
			settings: FieldFormatSettings{
				FormatType: PresetRegexpMapping,
				RegexpMapping: RegexpMappingSettings{
					Rules: []RegexpMappingRule{
						{
							Regexp:  `(\d+)`,
							Mapping: "numbers",
						},
					},
				},
			},
			value:        "123",
			expected:     "numbers",
			expectedType: types.LogExtractValueTypeText,
		},
		{
			name: "test_8",
			settings: FieldFormatSettings{
				FormatType: PresetAllmatch,
				TimeFormat: TimeFormatSettings{
					DateFormat: "2006-01-02 15:04:05",
				},
			},
			value:        "2025-01-01 12:00:00",
			expected:     time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			expectedType: types.LogExtractValueTypeDate,
		},
	}

	for _, test := range tests {
		if test.name != "test_4" {
			continue
		}
		t.Run(test.name, func(t *testing.T) {
			fieldFormat, err := NewFieldFormat(test.settings, test.expectedType)
			if err != nil {
				t.Fatalf("NewFieldFormat failed: %v", err)
			}
			actual, err := fieldFormat.Format(test.value)
			if err != nil {
				t.Fatalf("Format[%s] failed: %v", test.name, err)
			}
			if actual != test.expected {
				t.Fatalf("Format[%s] failed: expected %v, got %v", test.name, test.expected, actual)
			}
		})
	}
}
