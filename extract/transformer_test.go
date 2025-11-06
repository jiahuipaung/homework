package extract

import (
	"reflect"
	"testing"

	"github.com/flashcatcloud/fc-stash/transform/field_format"
	"github.com/flashcatcloud/fc-stash/types"
)

// 基本功能测试
func Test_Transformer(t *testing.T) {
	inputs := []struct {
		name     string
		ignore   bool
		origin   map[string]interface{}
		settings FieldTransformSettings
		expected map[string]interface{}
	}{
		{
			name:   "format_origin",
			ignore: false,
			origin: map[string]interface{}{"a": "123456"},
			settings: FieldTransformSettings{
				RuleType:    FieldTransformTypeFormat,
				OriginField: "a",
				TargetType:  types.LogExtractValueTypeLong,
				FormatSettings: field_format.FieldFormatSettings{
					FormatType: field_format.PresetRegexp,
					Regexp:     `(\d+)`,
				},
			},
			expected: map[string]interface{}{"a": int64(123456)},
		},
		{
			name:   "format_nested",
			ignore: false,
			origin: map[string]interface{}{"a": map[string]interface{}{"b": "123456"}},
			settings: FieldTransformSettings{
				RuleType:    FieldTransformTypeFormat,
				OriginField: "a.b",
				TargetType:  types.LogExtractValueTypeLong,
				FormatSettings: field_format.FieldFormatSettings{
					FormatType: field_format.PresetRegexp,
					Regexp:     `(\d+)`,
				},
			},
			expected: map[string]interface{}{"a": map[string]interface{}{"b": int64(123456)}},
		},
		{
			name:   "append_input_nested",
			ignore: false,
			origin: map[string]interface{}{"a": "123456"},
			settings: FieldTransformSettings{
				RuleType:    FieldTransformTypeAppend,
				TargetField: "b.c.d",
				TargetType:  types.LogExtractValueTypeLong,
				AppendSettings: AppendSettings{
					AppendType: FieldTransformAppendMethodInput,
					InputValue: "123456",
				},
			},
			expected: map[string]interface{}{"a": "123456", "b": map[string]interface{}{"c": map[string]interface{}{"d": int64(123456)}}},
		},
		{
			name:   "append_input_dict_mapping",
			ignore: false,
			origin: map[string]interface{}{"a": "123456"},
			settings: FieldTransformSettings{
				RuleType: FieldTransformTypeAppend,
				AppendSettings: AppendSettings{
					AppendType: FieldTransformAppendMethodLabelMapping,
					LabelMapping: LabelMappingSettings{
						ParentField: types.LogExtractJsonRoot,
						SchemaID:    1,
						Srcs:        []string{"a"},
						Dsts:        []string{"b", "c"},
					},
				},
			},
			expected: map[string]interface{}{"a": "123456", "b": "bxyz", "c": "cxyz"},
		},
		{
			name:   "append_input_dict_mapping_nested",
			ignore: false,
			origin: map[string]interface{}{"message": map[string]interface{}{"a": "123456"}},
			settings: FieldTransformSettings{
				RuleType: FieldTransformTypeAppend,
				AppendSettings: AppendSettings{
					AppendType: FieldTransformAppendMethodLabelMapping,
					LabelMapping: LabelMappingSettings{
						ParentField: "message",
						SchemaID:    1,
						Srcs:        []string{"a"},
						Dsts:        []string{"b", "c"},
					},
				},
			},
			expected: map[string]interface{}{"message": map[string]interface{}{"a": "123456", "b": "bxyz", "c": "cxyz"}},
		},
		{
			name:   "delete",
			ignore: false,
			origin: map[string]interface{}{"a": map[string]interface{}{"b": "123456", "c": "123456"}},
			settings: FieldTransformSettings{
				RuleType:    FieldTransformTypeDelete,
				OriginField: "a.b",
			},
			expected: map[string]interface{}{"a": map[string]interface{}{"c": "123456"}},
		},
		{
			name:   "clone",
			ignore: false,
			origin: map[string]interface{}{"a": map[string]interface{}{"b": "123456", "c": "123456"}},
			settings: FieldTransformSettings{
				RuleType:    FieldTransformTypeClone,
				OriginField: "a",
				TargetField: "a.copy",
			},
			expected: map[string]interface{}{"a": map[string]interface{}{"b": "123456", "c": "123456"}, "copy": map[string]interface{}{"b": "123456", "c": "123456"}},
		},
		{
			name:   "rename",
			ignore: false,
			origin: map[string]interface{}{"a": map[string]interface{}{"b": "123456", "c": "123456"}},
			settings: FieldTransformSettings{
				RuleType:    FieldTransformTypeRename,
				OriginField: "a.b",
				TargetField: "a.d",
			},
			expected: map[string]interface{}{"a": map[string]interface{}{"c": "123456", "d": "123456"}},
		},
		{
			name:   "rename_to_root",
			ignore: false,
			origin: map[string]interface{}{"a": map[string]interface{}{"b": "123456", "c": "123456"}, "copy": map[string]interface{}{"b": "123456", "c": "123456"}},
			settings: FieldTransformSettings{
				RuleType:    FieldTransformTypeRename,
				OriginField: "copy.b",
				TargetField: "__root__",
			},
			expected: map[string]interface{}{"a": map[string]interface{}{"b": "123456", "c": "123456"}, "b": "123456", "copy": map[string]interface{}{"c": "123456"}},
		},
		{
			name:   "rename_map_to_root",
			ignore: false,
			origin: map[string]interface{}{"a": map[string]interface{}{"b": "123456", "c": "123456"}, "copy": map[string]interface{}{"b": "123456", "c": "123456"}},
			settings: FieldTransformSettings{
				RuleType:    FieldTransformTypeRename,
				OriginField: "a",
				TargetField: "__root__",
			},
			expected: map[string]interface{}{"b": "123456", "c": "123456", "copy": map[string]interface{}{"b": "123456", "c": "123456"}},
		},
		{
			name:   "clone_and_format_origin",
			ignore: false,
			origin: map[string]interface{}{"a": "123456"},
			settings: FieldTransformSettings{
				RuleType:    FieldTransformTypeCloneAndFormat,
				OriginField: "a",
				TargetField: "b",
				TargetType:  types.LogExtractValueTypeLong,
				FormatSettings: field_format.FieldFormatSettings{
					FormatType: field_format.PresetRegexp,
					Regexp:     `(\d+)`,
				},
			},
			expected: map[string]interface{}{"a": "123456", "b": int64(123456)},
		},
		{
			name:   "clone_and_format_to_root",
			ignore: false,
			origin: map[string]interface{}{"a": "123456", "nested": map[string]interface{}{"b": "123456"}},
			settings: FieldTransformSettings{
				RuleType:    FieldTransformTypeCloneAndFormat,
				OriginField: "nested.b",
				TargetField: "__root__",
				TargetType:  types.LogExtractValueTypeLong,
				FormatSettings: field_format.FieldFormatSettings{
					FormatType: field_format.PresetRegexp,
					Regexp:     `(\d+)`,
				},
			},
			expected: map[string]interface{}{"a": "123456", "b": int64(123456), "nested": map[string]interface{}{"b": "123456"}},
		},
	}
	SetLabelMappingSchemas(map[int64]map[string]map[string]string{
		1: {"a=123456": {"b": "bxyz", "c": "cxyz"}},
	})
	for _, input := range inputs {
		if input.ignore {
			continue
		}
		fieldTransform, err := NewFieldTransform(input.settings)
		if err != nil {
			t.Errorf("test[%s] NewFieldTransform(%v) error: %v", input.name, input.settings, err)
			continue
		}
		if err := fieldTransform.Transform(input.origin); err != nil {
			t.Errorf("test[%s] FieldTransform.Transform(%v) error: %v", input.name, input.origin, err)
			continue
		}
		if !reflect.DeepEqual(input.origin, input.expected) {
			t.Errorf("test[%s] FieldTransform.Format(%v) = %v, want %v", input.name, input.origin, input.origin, input.expected)
		}
	}
}

// ignore_if_conflict/drop_if_not_found等flag的测试
func Test_Transformer_Flags(t *testing.T) {
	inputs := []struct {
		name          string
		ignore        bool
		origin        map[string]interface{}
		settings      FieldTransformSettings
		expectedError error
		expected      map[string]interface{}
	}{
		{
			name:   "format_drop_if_not_found",
			ignore: false,
			origin: map[string]interface{}{"a": "123456"},
			settings: FieldTransformSettings{
				RuleType:    FieldTransformTypeFormat,
				OriginField: "b",
				TargetType:  types.LogExtractValueTypeLong,
				FormatSettings: field_format.FieldFormatSettings{
					FormatType: field_format.PresetRegexp,
					Regexp:     `(\d+)`,
				},
				DropIfNotFound: true,
			},
			expectedError: ErrorOriginFieldNotFound,
			expected:      map[string]interface{}{"a": "123456"},
		},
		{
			name:   "append_ignore_if_conflict",
			ignore: false,
			origin: map[string]interface{}{"a": "123456"},
			settings: FieldTransformSettings{
				RuleType:    FieldTransformTypeAppend,
				TargetField: "a",
				AppendSettings: AppendSettings{
					AppendType: FieldTransformAppendMethodInput,
					InputValue: "123",
				},
				IgnoreIfConflict: true,
			},
			expected: map[string]interface{}{"a": "123456"},
		},
		{
			name:   "clone_ignore_if_conflict",
			ignore: true,
			origin: map[string]interface{}{"a": map[string]interface{}{"b": "123456", "c": "123456"}, "copy": "exist"},
			settings: FieldTransformSettings{
				RuleType:         FieldTransformTypeClone,
				OriginField:      "a",
				TargetField:      "copy",
				IgnoreIfConflict: true,
			},
			expected: map[string]interface{}{"a": map[string]interface{}{"b": "123456", "c": "123456"}, "copy": "exist"},
		},
		{
			name:   "clone_drop_if_not_found",
			ignore: false,
			origin: map[string]interface{}{"a": map[string]interface{}{"b": "123456", "c": "123456"}},
			settings: FieldTransformSettings{
				RuleType:       FieldTransformTypeClone,
				OriginField:    "a.x",
				TargetField:    "a.y",
				DropIfNotFound: true,
			},
			expectedError: ErrorOriginFieldNotFound,
			expected:      map[string]interface{}{"a": map[string]interface{}{"b": "123456", "c": "123456"}},
		},
		{
			name:   "clone_to_root_ignore_if_conflict",
			ignore: false,
			origin: map[string]interface{}{"a": map[string]interface{}{"b": "123456", "c": "123456"}, "b": "exist"},
			settings: FieldTransformSettings{
				RuleType:         FieldTransformTypeClone,
				OriginField:      "a",
				TargetField:      types.LogExtractJsonRoot,
				IgnoreIfConflict: true,
			},
			expected: map[string]interface{}{"a": map[string]interface{}{"b": "123456", "c": "123456"}, "b": "exist"},
		},
		{
			name:   "clone_and_format_to_root_ignore_if_conflict",
			ignore: false,
			origin: map[string]interface{}{"a": "123456", "nested": map[string]interface{}{"b": "123456"}, "b": "exist"},
			settings: FieldTransformSettings{
				RuleType:    FieldTransformTypeCloneAndFormat,
				OriginField: "nested.b",
				TargetField: "__root__",
				TargetType:  types.LogExtractValueTypeLong,
				FormatSettings: field_format.FieldFormatSettings{
					FormatType: field_format.PresetRegexp,
					Regexp:     `(\d+)`,
				},
				IgnoreIfConflict: true,
			},
			expected: map[string]interface{}{"a": "123456", "b": "exist", "nested": map[string]interface{}{"b": "123456"}},
		},
		{
			name:   "clone_and_format_to_root_drop_if_not_found",
			ignore: false,
			origin: map[string]interface{}{"a": "123456", "nested": map[string]interface{}{"b": "123456"}, "b": "exist"},
			settings: FieldTransformSettings{
				RuleType:    FieldTransformTypeCloneAndFormat,
				OriginField: "nested.x",
				TargetField: "__root__",
				TargetType:  types.LogExtractValueTypeLong,
				FormatSettings: field_format.FieldFormatSettings{
					FormatType: field_format.PresetRegexp,
					Regexp:     `(\d+)`,
				},
				DropIfNotFound: true,
			},
			expectedError: ErrorOriginFieldNotFound,
			expected:      map[string]interface{}{"a": "123456", "nested": map[string]interface{}{"b": "123456"}, "b": "exist"},
		},
	}
	for _, input := range inputs {
		if input.ignore {
			continue
		}
		fieldTransform, err := NewFieldTransform(input.settings)
		if err != nil {
			t.Errorf("test[%s] NewFieldTransform(%v) error: %v", input.name, input.settings, err)
			continue
		}
		if err := fieldTransform.Transform(input.origin); err != nil {
			if input.expectedError != nil {
				if err != input.expectedError {
					t.Errorf("test[%s] FieldTransform.Transform(%v) error: %v, want %v", input.name, input.origin, err, input.expectedError)
					continue
				}
			} else {
				t.Errorf("test[%s] FieldTransform.Transform(%v) error: %v", input.name, input.origin, err)
				continue
			}
		}
		if !reflect.DeepEqual(input.origin, input.expected) {
			t.Errorf("test[%s] FieldTransform.Format(%v) = %v, want %v", input.name, input.origin, input.origin, input.expected)
		}
	}
}
