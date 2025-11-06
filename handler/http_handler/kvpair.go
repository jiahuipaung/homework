package http_handler

import (
	"strings"

	"github.com/flashcatcloud/fc-stash/extract"
	"github.com/flashcatcloud/fc-stash/transform/field_format"
	"github.com/flashcatcloud/fc-stash/transform/filter"
	"github.com/flashcatcloud/fc-stash/transform/text_parser"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
	"gopkg.in/yaml.v2"
)

// http_handler 封装了网页中需要使用的方法
// 之所以单独封装, 是因为上层有两个项目需要使用, 而且这两个项目在data_source层面的实现是不同的
// 因此通过封装的中间层, 把差异屏蔽掉

// i18n的词典和映射规则
// 参考 https://github.com/toolkits/pkg/blob/master/i18n/i18n.go

const (
	LogExtractRuleTypeSubMatchAndGraft = "submatch_graft"
	LogExtractRuleTypeOrigin           = "origin" // 保持原状
)

type KeyPair struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	Placeholder  string `json:"placeholder,omitempty"` // 占位符
	BreakIfEmpty bool   `json:"break_if_empty"`        // 空则报错
	Tips         string `json:"tips,omitempty"`        // 说明
}

func GetLogEventSystemKeyPairs(lang string) map[string][]KeyPair {
	ret := make(map[string][]KeyPair)
	// pre_extract 即 text_parser
	ret["pre_extract"] = []KeyPair{
		{
			text_parser.PresetGonx,
			FormatKeyvaluePairByLang(lang, "logx_"+text_parser.PresetGonx),
			FormatKeyvaluePairByLang(lang, "logx_input_"+text_parser.PresetGonx),
			true,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+text_parser.PresetGonx),
		},
		/* grok性能太差, 暂时隐藏
		{
			filter.PresetGrok,
			FormatKeyvaluePairByLang(lang, "logx_"+filter.PresetGrok),
			FormatKeyvaluePairByLang(lang, "logx_input_"+filter.PresetGrok),
			true,
			// 特殊处理
			strings.Replace(FormatKeyvaluePairByLang(lang, "logx_desc_"+filter.PresetGrok), "NEWLINE", "\n", 1),
		},
		*/
		{
			text_parser.PresetRegexp,
			FormatKeyvaluePairByLang(lang, "logx_"+text_parser.PresetRegexp),
			FormatKeyvaluePairByLang(lang, "logx_input_"+text_parser.PresetRegexp+"_0"), // _0 是为了解决冲突
			true,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+text_parser.PresetRegexp),
		},
		{
			text_parser.PresetSplit,
			FormatKeyvaluePairByLang(lang, "logx_"+text_parser.PresetSplit),
			FormatKeyvaluePairByLang(lang, "logx_input_"+text_parser.PresetSplit),
			true,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+text_parser.PresetSplit),
		},
		{
			text_parser.PresetJson,
			FormatKeyvaluePairByLang(lang, "logx_"+text_parser.PresetJson),
			"{}",
			false, // 默认false
			FormatKeyvaluePairByLang(lang, "logx_desc_"+text_parser.PresetJson),
		},
		{
			text_parser.PresetKeyValue,
			FormatKeyvaluePairByLang(lang, "logx_"+text_parser.PresetKeyValue),
			FormatKeyvaluePairByLang(lang, "logx_input_"+text_parser.PresetKeyValue),
			true,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+text_parser.PresetKeyValue),
		},
		{
			text_parser.PresetOTELTraceSpans,
			FormatKeyvaluePairByLang(lang, "logx_"+text_parser.PresetOTELTraceSpans),
			"{}",
			false,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+text_parser.PresetOTELTraceSpans),
		},
	}
	// geoip可用
	if _, err := text_parser.NewGeoIPParser("", false); err == nil {
		ret["pre_extract"] = append(ret["pre_extract"], KeyPair{
			text_parser.PresetGeoIP,
			"geoip",
			"{}",
			false,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+text_parser.PresetGeoIP),
		})
	}
	ret["pre_function"] = []KeyPair{
		{
			filter.GlobalFuncStringContains,
			FormatKeyvaluePairByLang(lang, "logx_"+filter.GlobalFuncStringContains),
			FormatKeyvaluePairByLang(lang, "logx_input_text"),
			true, "",
		},
		{
			filter.GlobalFuncStringNotContains,
			FormatKeyvaluePairByLang(lang, "logx_"+filter.GlobalFuncStringNotContains),
			FormatKeyvaluePairByLang(lang, "logx_input_text"),
			true, "",
		},
		{
			filter.GlobalFuncStringMatch,
			FormatKeyvaluePairByLang(lang, "logx_"+filter.GlobalFuncStringMatch),
			FormatKeyvaluePairByLang(lang, "logx_input_regexp"),
			true, "",
		},
		{
			filter.GlobalFuncStringNotMatch,
			FormatKeyvaluePairByLang(lang, "logx_"+filter.GlobalFuncStringNotMatch),
			FormatKeyvaluePairByLang(lang, "logx_input_regexp"),
			true, "",
		},
		{
			filter.GlobalFuncStripAnsi,
			FormatKeyvaluePairByLang(lang, "logx_"+filter.GlobalFuncStripAnsi),
			"",
			false, "",
		},
		{
			filter.GlobalFuncFormatToJson,
			FormatKeyvaluePairByLang(lang, "logx_"+filter.GlobalFuncFormatToJson),
			"",
			false, "",
		},
		{
			filter.GlobalFuncDecompress,
			FormatKeyvaluePairByLang(lang, "logx_"+filter.GlobalFuncDecompress),
			"gzip",
			true, "",
		},
		{
			filter.GlobalFuncSplitJsonArray,
			FormatKeyvaluePairByLang(lang, "logx_"+filter.GlobalFuncSplitJsonArray),
			"",
			false, "",
		},
	}
	// 转换规则
	ret["transform_type"] = []KeyPair{
		{
			extract.FieldTransformTypeFormat,
			FormatKeyvaluePairByLang(lang, "logx_"+extract.FieldTransformTypeFormat),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+extract.FieldTransformTypeFormat),
		},
		{
			extract.FieldTransformTypeClone,
			FormatKeyvaluePairByLang(lang, "logx_"+extract.FieldTransformTypeClone),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+extract.FieldTransformTypeClone),
		},
		{
			extract.FieldTransformTypeRename,
			FormatKeyvaluePairByLang(lang, "logx_"+extract.FieldTransformTypeRename),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+extract.FieldTransformTypeRename),
		},
		{
			extract.FieldTransformTypeAppend,
			FormatKeyvaluePairByLang(lang, "logx_"+extract.FieldTransformTypeAppend),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+extract.FieldTransformTypeAppend),
		},
		{
			extract.FieldTransformTypeCloneAndFormat,
			FormatKeyvaluePairByLang(lang, "logx_"+extract.FieldTransformTypeCloneAndFormat),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+extract.FieldTransformTypeCloneAndFormat),
		},
		{
			extract.FieldTransformTypeDelete,
			FormatKeyvaluePairByLang(lang, "logx_"+extract.FieldTransformTypeDelete),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+extract.FieldTransformTypeDelete),
		},
	}
	ret["format_field_parser"] = []KeyPair{
		{
			field_format.PresetAllmatch,
			FormatKeyvaluePairByLang(lang, "logx_"+field_format.PresetAllmatch),
			"", false,
			"", // 不需要tips
		},
		{
			field_format.PresetRegexp,
			FormatKeyvaluePairByLang(lang, "logx_"+field_format.PresetRegexp),
			"", false,
			"",
		},
		{
			field_format.PresetDesensitize,
			FormatKeyvaluePairByLang(lang, "logx_"+field_format.PresetDesensitize),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+field_format.PresetDesensitize),
		},
		{
			field_format.PresetRegexpMapping,
			FormatKeyvaluePairByLang(lang, "logx_"+field_format.PresetRegexpMapping),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+field_format.PresetRegexpMapping),
		},
		{
			field_format.PresetUriPath,
			FormatKeyvaluePairByLang(lang, "logx_"+field_format.PresetUriPath),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+field_format.PresetUriPath),
		},
	}
	ret["desensitize_type"] = []KeyPair{
		{
			field_format.DesensitizeMethodPhone,
			FormatKeyvaluePairByLang(lang, "logx_desensitize_"+field_format.DesensitizeMethodPhone),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_desensitize_"+field_format.DesensitizeMethodPhone),
		},
		{
			field_format.DesensitizeMethodEmail,
			FormatKeyvaluePairByLang(lang, "logx_desensitize_"+field_format.DesensitizeMethodEmail),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_desensitize_"+field_format.DesensitizeMethodEmail),
		},
		{
			field_format.DesensitizeMethodIP,
			FormatKeyvaluePairByLang(lang, "logx_desensitize_"+field_format.DesensitizeMethodIP),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_desensitize_"+field_format.DesensitizeMethodIP),
		},
		{
			field_format.DesensitizeMethodBankCard,
			FormatKeyvaluePairByLang(lang, "logx_desensitize_"+field_format.DesensitizeMethodBankCard),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_desensitize_"+field_format.DesensitizeMethodBankCard),
		},
		{
			field_format.DesensitizeMethodIDCard,
			FormatKeyvaluePairByLang(lang, "logx_desensitize_"+field_format.DesensitizeMethodIDCard),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_desensitize_"+field_format.DesensitizeMethodIDCard),
		},
		{
			field_format.DesensitizeMethodAccessKey,
			FormatKeyvaluePairByLang(lang, "logx_desensitize_"+field_format.DesensitizeMethodAccessKey),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_desensitize_"+field_format.DesensitizeMethodAccessKey),
		},
		{
			field_format.DesensitizeMethodKeepFirstLast,
			FormatKeyvaluePairByLang(lang, "logx_desensitize_"+field_format.DesensitizeMethodKeepFirstLast),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_desensitize_"+field_format.DesensitizeMethodKeepFirstLast),
		},
		{
			field_format.DesensitizeMethodKeepLastFour,
			FormatKeyvaluePairByLang(lang, "logx_desensitize_"+field_format.DesensitizeMethodKeepLastFour),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_desensitize_"+field_format.DesensitizeMethodKeepLastFour),
		},
		{
			field_format.DesensitizeMethodMd5,
			FormatKeyvaluePairByLang(lang, "logx_desensitize_"+field_format.DesensitizeMethodMd5),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_desensitize_"+field_format.DesensitizeMethodMd5),
		},
		{
			field_format.DesensitizeMethodCustom,
			FormatKeyvaluePairByLang(lang, "logx_desensitize_"+field_format.DesensitizeMethodCustom),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_desensitize_"+field_format.DesensitizeMethodCustom),
		},
	}
	ret["extract_type"] = []KeyPair{
		{
			extract.LogExtractRuleTypeSubMatch,
			FormatKeyvaluePairByLang(lang, "logx_"+extract.LogExtractRuleTypeSubMatch),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+extract.LogExtractRuleTypeSubMatch),
		},
		{
			extract.LogExtractRuleTypeGraft,
			FormatKeyvaluePairByLang(lang, "logx_"+extract.LogExtractRuleTypeGraft),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+extract.LogExtractRuleTypeGraft),
		},
		{
			LogExtractRuleTypeSubMatchAndGraft,
			FormatKeyvaluePairByLang(lang, "logx_"+LogExtractRuleTypeSubMatchAndGraft),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+LogExtractRuleTypeSubMatchAndGraft),
		},
		{
			extract.LogExtractRuleTypeAppend,
			FormatKeyvaluePairByLang(lang, "logx_"+extract.LogExtractRuleTypeAppend),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+extract.LogExtractRuleTypeAppend),
		},
		{
			extract.LogExtractRuleTypeRemove,
			FormatKeyvaluePairByLang(lang, "logx_"+extract.LogExtractRuleTypeRemove),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+extract.LogExtractRuleTypeRemove),
		},
		{
			LogExtractRuleTypeOrigin,
			FormatKeyvaluePairByLang(lang, "logx_"+LogExtractRuleTypeOrigin),
			"", false,
			FormatKeyvaluePairByLang(lang, "logx_desc_"+LogExtractRuleTypeOrigin),
		},
	}
	// Deprecated
	ret["preset_filter"] = []KeyPair{
		{Key: field_format.PresetAllmatch, Name: FormatKeyvaluePairByLang(lang, field_format.PresetAllmatch)},
		{Key: field_format.PresetUriPath, Name: FormatKeyvaluePairByLang(lang, field_format.PresetUriPath)},
	}

	ret["value_type"] = []KeyPair{
		{Key: extract.LogExtractValueTypeText, Name: FormatKeyvaluePairByLang(lang, extract.LogExtractValueTypeText)},
		{Key: extract.LogExtractValueTypeFloat, Name: FormatKeyvaluePairByLang(lang, extract.LogExtractValueTypeFloat)},
		{Key: extract.LogExtractValueTypeDate, Name: FormatKeyvaluePairByLang(lang, extract.LogExtractValueTypeDate)},
		{Key: extract.LogExtractValueTypeLong, Name: FormatKeyvaluePairByLang(lang, extract.LogExtractValueTypeLong)},
		{Key: extract.LogExtractValueTypeObject, Name: FormatKeyvaluePairByLang(lang, extract.LogExtractValueTypeObject)},
		{Key: extract.LogExtractValueTypeArray, Name: FormatKeyvaluePairByLang(lang, extract.LogExtractValueTypeArray)},
	}

	ret["index_suffix_format"] = []KeyPair{
		{Key: "default", Name: FormatKeyvaluePairByLang(lang, "suffix_default")},
		{Key: "none", Name: FormatKeyvaluePairByLang(lang, "suffix_none")},
		{Key: "hourly", Name: FormatKeyvaluePairByLang(lang, "suffix_hourly")},
		{Key: "daily", Name: FormatKeyvaluePairByLang(lang, "suffix_daily")},
		{Key: "weekly", Name: FormatKeyvaluePairByLang(lang, "suffix_weekly")},
	}
	ret["retention_duration"] = []KeyPair{
		{Key: "default", Name: FormatKeyvaluePairByLang(lang, "suffix_default")},
		{Key: "0", Name: FormatKeyvaluePairByLang(lang, "retention_0")},
		{Key: "2h", Name: FormatKeyvaluePairByLang(lang, "retention_2h")},
		{Key: "1d", Name: FormatKeyvaluePairByLang(lang, "retention_1d")},
		{Key: "2d", Name: FormatKeyvaluePairByLang(lang, "retention_2d")},
		{Key: "5d", Name: FormatKeyvaluePairByLang(lang, "retention_5d")},
		{Key: "1w", Name: FormatKeyvaluePairByLang(lang, "retention_1w")},
		{Key: "2w", Name: FormatKeyvaluePairByLang(lang, "retention_2w")},
		{Key: "30d", Name: FormatKeyvaluePairByLang(lang, "retention_30d")},
		{Key: "92d", Name: FormatKeyvaluePairByLang(lang, "retention_92d")},
		{Key: "185d", Name: FormatKeyvaluePairByLang(lang, "retention_185d")},
	}
	return ret
}

// 日志提取的内置词表, 统一处理
var (
	dict_printers = make(map[string]*message.Printer)
	dict_langs    = make(map[string]map[string]string)
)

func init() {
	dict := make(map[string]map[string]string)

	err := yaml.Unmarshal([]byte(i18ndict), &dict)
	if err == nil {
		RegisterI18nDict(dict)
	}
}

func FormatKeyvaluePairByLang(lang string, key string) string {
	abbr, _ := langAbbrAndTag(lang)
	if _, found := dict_langs[abbr]; !found {
		return key
	}
	return dict_printers[abbr].Sprintf(key)
}

func RegisterI18nDict(dict map[string]map[string]string) {
	dict_printers = make(map[string]*message.Printer)
	dict_langs = make(map[string]map[string]string)
	reversed := make(map[string]map[string]string)
	for key, _dict := range dict {
		for lang, value := range _dict {
			if _, found := reversed[lang]; !found {
				reversed[lang] = make(map[string]string)
			}
			reversed[lang][key] = value
		}
	}
	for lang, _dict := range reversed {
		abbr, tag := langAbbrAndTag(lang)
		cata := catalog.NewBuilder()
		for k, v := range _dict {
			cata.SetString(tag, k, v)
		}
		if _, found := dict_langs[abbr]; !found {
			dict_langs[abbr] = _dict
		}
		dict_printers[abbr] = message.NewPrinter(tag, message.Catalog(cata))
	}
}

// 默认返回中文
func langAbbrAndTag(l string) (string, language.Tag) {
	switch strings.ToLower(l) {
	case "en", "en_us":
		return "en", language.English

	case "ja_jp", "jp":
		return "jp", language.Japanese

	case "ko_kr", "kr":
		return "kr", language.Korean

	case "zh_hk", "zh_tw", "hk", "tw":
		return "hk", language.TraditionalChinese

	default:
		return "zh", language.Chinese
	}
}
