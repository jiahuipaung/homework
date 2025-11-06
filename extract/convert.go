package extract

import (
	"github.com/flashcatcloud/fc-stash/transform/field_format"
)

/*
// 根据benchmark结果, 新版本与旧版本性能相差不大, 不会引入性能瓶颈
goos: darwin
goarch: amd64
pkg: github.com/flashcatcloud/fc-stash/extract
cpu: VirtualApple @ 2.50GHz
Benchmark_JsonPruneV2_Normal-8                     25578             46660 ns/op           14369 B/op        177 allocs/op
Benchmark_JsonPruneV2_OverwriteOrigin-8            25854             46334 ns/op           14679 B/op        180 allocs/op
Benchmark_LogTransformer_Normal-8                  30091             39563 ns/op           14388 B/op        179 allocs/op
Benchmark_LogTransformer_OverwriteOrigin-8         30210             39559 ns/op           14364 B/op        179 allocs/op
*/

func ConvertJsonPruneV2ToTransformSettings(jsonPruneV2 *JsonPruneV2) (*TransformSettings, error) {
	transformer := new(TransformSettings)
	transformer.PreFunction = jsonPruneV2.PreFunction
	transformer.PrefixMatch = jsonPruneV2.PrefixMatch
	transformer.PreExtract = jsonPruneV2.PreExtract
	transformer.MultiPreExtract = jsonPruneV2.MultiPreExtract
	transformer.SkipExtractError = jsonPruneV2.SkipExtractError
	transformer.OverwriteOrigin = jsonPruneV2.OverwriteOrigin
	uniqueDelete := make(map[string]struct{})
	var renames []FieldTransformSettings
	for _, fieldExtract := range jsonPruneV2.Fields {
		fieldTransformSettings, todeletes := ConvertFieldExtractToFieldTransformSettings(fieldExtract)
		if fieldTransformSettings.RuleType == FieldTransformTypeRename {
			renames = append(renames, *fieldTransformSettings)
		} else {
			transformer.FieldsTransform = append(transformer.FieldsTransform, *fieldTransformSettings)
		}
		for _, delete := range todeletes {
			if _, ok := uniqueDelete[delete]; ok {
				continue
			}
			uniqueDelete[delete] = struct{}{}
		}
	}
	for delete := range uniqueDelete {
		transformer.FieldsTransform = append(transformer.FieldsTransform, FieldTransformSettings{
			RuleType:    FieldTransformTypeDelete,
			OriginField: delete,
		})
	}
	// 按照旧版的逻辑, rename的需要最后执行
	transformer.FieldsTransform = append(transformer.FieldsTransform, renames...)
	return transformer, nil
}

// 将旧的提取规则转换为新的提取规则
// 主要用于前端展示和后端兼容
func ConvertFieldExtractToFieldTransformSettings(fieldExtract *FieldExtract) (*FieldTransformSettings, []string) {
	if fieldExtract == nil {
		return &FieldTransformSettings{}, []string{}
	}
	ret := new(FieldTransformSettings)
	var todeletes []string
	switch fieldExtract.RuleType {
	case LogExtractRuleTypeAppend: // 对应的是append
		ret.RuleType = FieldTransformTypeAppend
		ret.TargetField = fieldExtract.Key
		ret.TargetType = fieldExtract.ValueType
		ret.AppendSettings.AppendType = FieldTransformAppendMethodInput
		ret.AppendSettings.InputValue = fieldExtract.Extract.AppendValue
		ret.DropIfNotFound = fieldExtract.Required

	case LogExtractRuleTypeSubMatch:
		if fieldExtract.OriginField == fieldExtract.Key { // 对应的是format
			ret.RuleType = FieldTransformTypeFormat
		} else { // clone_and_format(确保原字段保留), 最后再把原始字段删掉
			todeletes = append(todeletes, fieldExtract.OriginField)
			ret.RuleType = FieldTransformTypeCloneAndFormat
			ret.TargetField = fieldExtract.Key
		}
		ret.OriginField = fieldExtract.OriginField
		ret.DropIfNotFound = fieldExtract.Required
		ret.TargetType = fieldExtract.ValueType
		if ret.TargetType == LogExtractValueTypeDate {
			ret.FormatSettings.TimeFormat.DateFormat = field_format.ConvertGolangDateFormatToLinux(fieldExtract.Extract.DateFormat)
			ret.FormatSettings.TimeFormat.DateLocation = fieldExtract.Extract.DateLocation
			ret.FormatSettings.TimeFormat.ConvertToUTC8 = fieldExtract.Extract.DateFixUTCOffset == field_format.ConvertToUTC8Offset
		}
		if len(fieldExtract.Extract.PresetFilter) > 0 {
			ret.FormatSettings.FormatType = fieldExtract.Extract.PresetFilter
		} else if len(fieldExtract.Extract.Regexp) > 0 {
			ret.FormatSettings.FormatType = field_format.PresetRegexp
			ret.FormatSettings.Regexp = fieldExtract.Extract.Regexp
		} else {
			ret.FormatSettings.FormatType = field_format.PresetAllmatch
		}

	case LogExtractRuleTypeRemove: // 对应的是delete
		ret.RuleType = FieldTransformTypeDelete
		ret.OriginField = fieldExtract.OriginField

	case LogExtractRuleTypeGraft: // 对应的是rename
		ret.RuleType = FieldTransformTypeRename
		ret.OriginField = fieldExtract.OriginField
		ret.TargetField = fieldExtract.Key

	}
	return ret, todeletes
}
