package extract

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/flashcatcloud/fc-stash/map_iterator"
	"github.com/flashcatcloud/fc-stash/transform/field_format"
	"github.com/flashcatcloud/fc-stash/transform/filter"
	"github.com/flashcatcloud/fc-stash/transform/text_parser"
	"github.com/flashcatcloud/fc-stash/types"
)

const (
	LogTransformFailure = "log_transform_failure"
)

// 日志转换器, 用于将日志转换为其他格式
// 复制JsonPruneV2的逻辑, 并进行部分重构
type LogTransformer struct {
	UUID       string
	IsCompiled bool
	Settings   *TransformSettings
	// 预处理
	prefixMatch     map[string]struct{} `json:"-"` // PrefixMatch的编译结果
	textParser      []*TextParser       `json:"-"` // 文本解析器, 与.PreExtract中的字段一一对应
	textParserCount map[string]int      `json:"-"` // 文本解析器计数, 同一个字段可以有多个解析器, 任意一个执行成功, 则不执行下一个
	fieldTransform  []*FieldTransform   `json:"-"` // 字段解析器, 与.FieldsTransform中的字段一一对应
}

type TextParser struct {
	field  string
	mode   string
	ignore bool // 是否忽略, 比如geoip parser
	path   []string
	parser types.TextParser
}

type FieldTransform struct {
	settings    FieldTransformSettings
	originPath  []string                  // 源字段的父节点
	originField string                    // 源字段
	targetPath  []string                  // 目标字段的父节点
	targetField string                    // 目标字段
	formatter   *field_format.FieldFormat // rule_type=format/clone_and_format时使用
	mapping     *LabelMapping             // rule_type=append时使用
}

func (t *LogTransformer) Init(ctx context.Context) error {
	return t.Compile()
}

func (t *LogTransformer) GetUUID() string {
	return t.UUID
}

func (t *LogTransformer) Compile(forceOrigin ...bool) error {
	// 避免重复执行
	if t.IsCompiled {
		return nil
	}
	// 筛选和预处理, 对KAFKA消息做过滤
	if len(t.Settings.PreFunction) > 0 {
		for _, pfunc := range t.Settings.PreFunction {
			if err := pfunc.Compile(); err != nil {
				return err
			}
		}
	}
	// 对第一层做剪枝, 简单处理
	if len(t.Settings.PrefixMatch) > 0 {
		t.prefixMatch = make(map[string]struct{})
		for _, filter := range t.Settings.PrefixMatch {
			if len(filter) == 0 {
				continue
			}
			t.prefixMatch[filter] = struct{}{}
		}
	}
	// 复杂文本处理, 把字符串转换成map[string]interface{}
	if len(t.Settings.PreExtract) > 0 {
		t.textParser = make([]*TextParser, len(t.Settings.PreExtract))
		t.textParserCount = make(map[string]int)
		for i, pre := range t.Settings.PreExtract {
			parser, err := text_parser.NewPresetTextParser(pre.Mode, pre.Format, false)
			if err != nil {
				// geoip parser 忽略错误
				if pre.Mode != text_parser.PresetGeoIP {
					return errors.New("field[" + pre.Field + "] invalid parser format:" + err.Error())
				}
			}
			// parser
			t.textParserCount[pre.Field]++
			t.textParser[i] = &TextParser{
				field:  pre.Field,
				mode:   pre.Mode,
				ignore: err != nil,                    // err != nil 代表是geoip parser, 下文使用时忽略这个textParser
				path:   strings.Split(pre.Field, "."), // 路径
				parser: parser,
			}
		}
	}
	// 字段转换, 根据配置的规则, 对某个字段进行转换, 以及新增字段、克隆字段、删除字段、重命名字段
	if len(t.Settings.FieldsTransform) > 0 {
		t.fieldTransform = make([]*FieldTransform, len(t.Settings.FieldsTransform))
		for i, trans := range t.Settings.FieldsTransform {
			fieldTransform, err := NewFieldTransform(trans)
			if err != nil {
				return errors.New("field[" + trans.OriginField + "] invalid field transform:" + err.Error())
			}
			t.fieldTransform[i] = fieldTransform
		}
	}

	t.IsCompiled = true
	return nil
}

func (t *LogTransformer) Handle(ctx context.Context, e *types.LogEvent) (types.ExtractedLog, error) {
	if e == nil || e.Message == nil {
		return nil, errors.New("nil event or empty message")
	}
	// e.Message 只有两种类型, string和map[string]interface{}
	// 1) string类型, 按照原来的逻辑处理
	// 2) map[string]interface{}, 说明不存在preFunction(前置处理时校验过)
	var extracted map[string]interface{}
	var err error
	// deep copy, 多个extractor复用
	message, ok := e.Message.(string)
	if ok {
		if len(t.Settings.PreFunction) > 0 {
			var ok bool
			message, ok = filter.StringPreExtractHandle(message, t.Settings.PreFunction, false)
			if !ok {
				// 不返回错误, 直接返回空
				return nil, nil
			}
		}
		extracted, err = t.OriginJsonLogPrune(message)
	} else {
		_level0, ok := e.Message.(map[string]interface{})
		if !ok {
			return nil, errors.New("invalid message type")
		}
		// deep copy
		level0 := make(map[string]interface{})
		for k, v := range _level0 {
			level0[k] = v
		}
		extracted, err = t.OriginJsonLogPruneWithMap(level0)
	}
	// 一旦某一步执行失败, 则直接退出
	if err != nil {
		return nil, err
	}
	// 如果extracted为空, 则直接返回空
	if len(extracted) == 0 {
		return nil, nil
	}
	return extracted, nil
}

func (t *LogTransformer) OriginJsonLogPrune(origin string) (ret map[string]interface{}, err error) {
	if !t.IsCompiled {
		return nil, errors.New("transformer not compiled")
	}
	// 输入是string, 需要转换成map[string]interface{}
	level0 := make(map[string]interface{})
	if err = json.Unmarshal([]byte(origin), &level0); err != nil {
		err = ErrorInvalidJsonString
		return
	}

	return t.OriginJsonLogPruneWithMap(level0)
}

// 输入是map[string]interface{}, 直接对map字段进行处理
func (t *LogTransformer) OriginJsonLogPruneWithMap(level0 map[string]interface{}) (ret map[string]interface{}, err error) {
	if !t.IsCompiled {
		return nil, errors.New("transformer not compiled")
	}
	// 特殊的前置处理, key-value的过滤
	// 在KAFKA input中无法处理, 需要转换成map[string]interface{}后处理
	if len(t.Settings.PreFunction) > 0 {
		for _, field := range t.Settings.PreFunction {
			if field.Mode == filter.GlobalFuncKeyValueMatch {
				if !field.KvMatchOrNotMatch(level0) {
					return nil, nil
				}
			} else if field.Mode == filter.GlobalFuncKeyValueNotMatch {
				if field.KvMatchOrNotMatch(level0) {
					return nil, nil
				}
			}
		}
	}

	// 对第一层做剪枝, 简单处理
	for key := range level0 {
		// 如果 prefixMatch 为空, 则全提取
		if len(t.prefixMatch) > 0 {
			if _, found := t.prefixMatch[key]; !found {
				delete(level0, key)
			}
		}
	}
	// 复杂文本处理, 把字符串转换成map[string]interface{}
	if len(t.Settings.PreExtract) > 0 {
		if err := t.TextTransform(level0); err != nil {
			return level0, err
		}
	}
	// 字段转换, 根据配置的规则, 对某个字段进行转换, 以及新增字段、克隆字段、删除字段、重命名字段
	if len(t.Settings.FieldsTransform) > 0 {
		if err := t.FieldsTransform(level0); err != nil {
			return level0, err
		}
	}
	return level0, nil
}

// textParser, 复杂文本提取
func (t *LogTransformer) TextTransform(m map[string]interface{}) error {
	if len(t.Settings.PreExtract) == 0 {
		return nil
	}
	var multiParserCount map[string]int
	for _, textParser := range t.textParser {
		if textParser.ignore {
			continue
		}
		if len(textParser.path) == 0 {
			continue
		}
		value, found := map_iterator.FindByPath(m, textParser.path)
		if !found {
			continue
		}
		vstr, ok := value.(string)
		if !ok {
			err := errors.New("field[" + textParser.field + "] is not a string")
			t.addErrorTags(m, err)
			if !t.Settings.SkipExtractError {
				return err
			}
			continue
		}
		// 如果同一个字段有多个解析器, 那么需要做两个判断
		// 1. 任意一个解析器执行成功后, 其他的就不执行了
		// 2. 如果所有解析器都执行失败, 则记录错误, 记录最后一条即可
		hasMultiParser := false
		if multi, ok := t.textParserCount[textParser.field]; ok && multi > 1 {
			hasMultiParser = true
			if len(multiParserCount) == 0 {
				multiParserCount = make(map[string]int)
			}
			if _, found := multiParserCount[textParser.field]; !found {
				multiParserCount[textParser.field] = multi
			}
		}
		var vmap map[string]interface{}
		var err error
		if hasMultiParser {
			if multiParserCount[textParser.field] > 0 {
				vmap, err = textParser.parser.Parse(vstr)
				multiParserCount[textParser.field]--
				if err != nil {
					// 如果计数为0, 说明前面的都执行失败了, 记录错误
					if multiParserCount[textParser.field] == 0 {
						t.addErrorTags(m, err)
						if !t.Settings.SkipExtractError {
							return err
						}
					}
					// 只要不return, 说明还有其他解析器, 继续执行
					continue

				} else if len(vmap) > 0 {
					// 置为0, 确保其他解析器不再执行
					multiParserCount[textParser.field] = 0
				}
			}
		} else {
			vmap, err = textParser.parser.Parse(vstr)
			if err != nil {
				t.addErrorTags(m, err)
				if !t.Settings.SkipExtractError {
					return err
				}
				continue
			}
		}
		overwriteOrigin := t.Settings.OverwriteOrigin
		// JSON反序列化, 直接覆盖原字段
		if textParser.mode == text_parser.PresetJson {
			overwriteOrigin = true
		}
		var rewriteMap map[string]interface{}
		if len(textParser.path) == 1 {
			rewriteMap = m
		} else {
			valueMap, found := map_iterator.FindByPath(m, textParser.path[:len(textParser.path)-1])
			if !found {
				err := errors.New("overwrite field failed, field[" + strings.Join(textParser.path[:len(textParser.path)-1], ".") + "] not found")
				t.addErrorTags(m, err)
				if !t.Settings.SkipExtractError {
					return err
				}
				continue
			}
			rewriteMap, ok = valueMap.(map[string]interface{})
			if !ok {
				err := errors.New("overwrite field failed, field[" + strings.Join(textParser.path[:len(textParser.path)-1], ".") + "] is not a map")
				t.addErrorTags(m, err)
				if !t.Settings.SkipExtractError {
					return err
				}
				continue
			}
		}
		rewriteTarget := textParser.path[len(textParser.path)-1]
		if overwriteOrigin {
			rewriteMap[rewriteTarget] = vmap
		} else {
			for k, v := range vmap {
				key := k
				// split parser统一加上前缀
				if textParser.mode == text_parser.PresetSplit {
					key = rewriteTarget + "_" + text_parser.PresetSplit + "_" + k
				}
				// 如果子串中存在与原字段同名, 则加一层前缀
				if _, found := rewriteMap[key]; found {
					key = rewriteTarget + "_" + textParser.mode + "_" + k
				}
				rewriteMap[key] = v
			}
		}
	}
	return nil
}

// fieldParser, 单个字段处理
// 1. 根据rule_type来处理, 不同的rule_type对应的逻辑不同
// 2. 每个rule_type写一段函数, 便于扩展
func (t *LogTransformer) FieldsTransform(m map[string]interface{}) error {
	if len(t.Settings.FieldsTransform) == 0 {
		return nil
	}
	for _, fieldTransform := range t.fieldTransform {
		if err := fieldTransform.Transform(m); err != nil {
			t.addErrorTags(m, err)
			// 如果跳过错误, 则继续处理下一个字段
			if !t.Settings.SkipExtractError {
				return err
			}
			continue
		}
	}
	return nil
}

func (t *LogTransformer) addErrorTags(origin map[string]interface{}, err error) error {
	if err == nil {
		return errors.New("get FailInformation error")
	}
	errStr := err.Error()
	if errStr == "" {
		return errors.New("get FailInformation error")
	}
	origin[LogTransformFailure] = errStr
	return nil
}
