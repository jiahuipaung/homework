package extract

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/flashcatcloud/fc-stash/transform/filter"
	"github.com/flashcatcloud/fc-stash/transform/text_parser"
	"github.com/flashcatcloud/fc-stash/types"
)

// 与V1的区别是, PreExtract的结果, 放到原始key的同级, 不再保留__origin__逻辑
// json反序列化除外
// jsonFlatten() 函数不同, 其他逻辑相同
// 新版日志提取中使用, 旧版后续不再维护
type JsonPruneV2 struct {
	UUID             string                        `json:"-"`                  // 用于配置更新
	IsCompiled       bool                          `json:"-"`                  //
	PreFunction      []*filter.StringPreFunction   `json:"pre_function"`       // 前置处理
	PrefixMatch      []string                      `json:"prefix_match"`       // 只支持第一层的过滤, 不做复杂的嵌套
	PreExtract       []*JsonPreSourceExtract       `json:"pre_extract"`        // 预提取
	MultiPreExtract  bool                          `json:"multi_pre_extract"`  // 单字段执执行多次, 如果有
	Fields           []*FieldExtract               `json:"fields"`             // 提取规则
	SkipExtractError bool                          `json:"skip_extract_error"` // 跳过单字段多规则中的解析失败的规则
	OverwriteOrigin  bool                          `json:"overwrite_origin"`   // 是否要覆盖原字段, json直接覆盖
	pruneIndex       map[string][]*FieldExtract    `json:"-"`                  // map索引
	prunePrefixIndex map[string]struct{}           `json:"-"`                  // 如果不在这个列表中, JsonPruneV2()中可以不遍历
	required         int                           `json:"-"`                  //
	prefixMatch      map[string]struct{}           `json:"-"`                  // PrefixMatch的编译结果
	preParser        map[string][]types.TextParser `json:"-"`                  //
	preParserPrefix  map[string]struct{}           `json:"-"`                  // 如果不在这个列表中, jsonFlatten中可以不遍历
}

func (le *JsonPruneV2) Init(ctx context.Context) error {
	return le.Compile()
}

func (le *JsonPruneV2) GetUUID() string {
	return le.UUID
}

func (le *JsonPruneV2) Compile(forceOrigin ...bool) error {
	// 避免重复执行
	if le.IsCompiled {
		return nil
	}
	if len(le.PreFunction) > 0 {
		for _, pfunc := range le.PreFunction {
			if err := pfunc.Compile(); err != nil {
				return err
			}
		}
	}
	if len(le.PreExtract) > 0 {
		le.preParser = make(map[string][]types.TextParser)
		le.preParserPrefix = make(map[string]struct{})
		fieldCount := make(map[string]int)

		//初始化每个字段的规则数
		for _, extract := range le.PreExtract {
			fieldCount[extract.Field]++
		}
		for field, count := range fieldCount {
			le.preParser[field] = make([]types.TextParser, count)
		}
		for _, extract := range le.PreExtract {
			fieldCount[extract.Field] = 0
		}

		for _, pre := range le.PreExtract {
			// 旧版配置, 兼容一下, 直接丢掉, 不再处理
			if pre.Mode == filter.GlobalFuncStripAnsi {
				continue
			}
			if err := pre.Compile(false); err != nil {
				// geoip parser 忽略错误
				if pre.Mode == text_parser.PresetGeoIP {
					continue
				}
				return err
			}
			if pre.parser == nil {
				continue
			}
			if pre.Mode == text_parser.PresetSplit {
				// not forceOrigin, 代表是stash阶段
				// 设置split的下标提取规则, 只提取"需要的"字段
				if !(len(forceOrigin) > 0 && forceOrigin[0]) {
					var splitindex []int
					for _, field := range le.Fields {
						if strings.HasPrefix(field.OriginField, pre.Field+"_"+text_parser.PresetSplit+"_") {
							idx, err := strconv.Atoi(strings.TrimPrefix(field.OriginField, pre.Field+"_"+text_parser.PresetSplit+"_"))
							if err == nil && idx >= 0 {
								splitindex = append(splitindex, idx)
							}
						}
					}
					// 如果没有设置任意规则, 则全部透传
					if len(splitindex) > 0 {
						pre.parser.(*text_parser.SplitParser).SetRemainIndex(splitindex)
					}
				}
			}
			le.preParser[pre.Field][fieldCount[pre.Field]] = pre.parser
			fieldCount[pre.Field]++
			sources := strings.Split(pre.Field, ".")
			for i := 0; i < len(sources); i++ { // preParser的父节点, "" 空字符串代表的是root
				le.preParserPrefix[strings.Join(sources[0:i], ".")] = struct{}{}
			}
		}
	}
	if len(le.PrefixMatch) > 0 {
		le.prefixMatch = make(map[string]struct{})
		for _, filter := range le.PrefixMatch {
			if len(filter) == 0 {
				continue
			}
			le.prefixMatch[filter] = struct{}{}
		}
	}
	// 没有fields提取则全部透传
	le.pruneIndex = make(map[string][]*FieldExtract)
	le.prunePrefixIndex = make(map[string]struct{})
	removeBl := make(map[string]struct{})
	for _, field := range le.Fields {
		if field.RuleType != LogExtractRuleTypeRemove {
			removeBl[field.Key] = struct{}{}
		}
	}
	for _, field := range le.Fields { // 按照fields配置的顺序生效
		// 如果要删除的字段是"提取/移动/新增"规则的目标字段, 那么该删除规则不需要执行(待另一个规则覆盖即可)
		if field.RuleType == LogExtractRuleTypeRemove {
			if _, found := removeBl[field.OriginField]; found {
				continue
			}
		}
		if err := field.Compile(); err != nil {
			return err
		}
		if field.Required {
			le.required++
		}
		if len(field.source) > 0 {
			sources := strings.Split(field.source, ".")
			for i := 1; i < len(sources); i++ { // 最后一段丢掉
				le.prunePrefixIndex[strings.Join(sources[0:i], ".")] = struct{}{}
			}
		}
		le.pruneIndex[field.source] = append(le.pruneIndex[field.source], field)
	}
	le.IsCompiled = true
	return nil
}

func (le *JsonPruneV2) Handle(ctx context.Context, e *types.LogEvent) (types.ExtractedLog, error) {
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
		if len(le.PreFunction) > 0 {
			var ok bool
			message, ok = filter.StringPreExtractHandle(message, le.PreFunction, false)
			if !ok {
				// 不返回错误, 直接返回空
				return nil, nil
			}
		}
		extracted, err = le.OriginJsonLogPrune(message)
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
		extracted, err = le.OriginJsonLogPruneWithMap(level0)
	}
	// 一旦某一步执行失败, 则直接退出
	if err != nil {
		return nil, err
	}
	// disable
	if len(extracted) == 0 {
		return nil, nil
	}
	return extracted, nil
}

// 简化OriginJsonLogFlatten()逻辑, 非必须的字段直接过滤
func (je *JsonPruneV2) OriginJsonLogPrune(origin string) (ret map[string]interface{}, err error) {
	if !je.IsCompiled {
		return nil, errors.New("rules not compiled")
	}
	level0 := make(map[string]interface{})
	if err = json.Unmarshal([]byte(origin), &level0); err != nil {
		err = ErrorInvalidJsonString
		return
	}

	//Key-Value对的匹配与排除
	if len(je.PreFunction) > 0 {
		for _, field := range je.PreFunction {
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

	// 对第一层做过滤, 不支持更多层
	for key := range level0 {
		// 如果 prefixMatch 为空, 则全提取
		if len(je.prefixMatch) > 0 {
			if _, found := je.prefixMatch[key]; !found {
				delete(level0, key)
			}
		}
	}
	if err := je.jsonFlatten(level0, ""); err != nil {
		je.addErrorTags(level0, err)
		return level0, err
	}
	must := int32(0)
	if len(je.pruneIndex) > 0 {
		if err := je.JsonPrune(level0, level0, "", &must); err != nil {
			return nil, err
		}
	}
	if int(must) < je.required {
		return nil, errors.New("one or more required fields missing")
	}
	return level0, nil
}
func (je *JsonPruneV2) OriginJsonLogPruneWithMap(level0 map[string]interface{}) (ret map[string]interface{}, err error) {
	if !je.IsCompiled {
		return nil, errors.New("rules not compiled")
	}
	// 对第一层做过滤, 不支持更多层
	for key := range level0 {
		// 如果 prefixMatch 为空, 则全提取
		if len(je.prefixMatch) > 0 {
			if _, found := je.prefixMatch[key]; !found {
				delete(level0, key)
			}
		}
	}
	if err := je.jsonFlatten(level0, ""); err != nil {
		je.addErrorTags(level0, err)
		return level0, err
	}
	must := int32(0)
	if len(je.pruneIndex) > 0 {
		if err := je.JsonPrune(level0, level0, "", &must); err != nil {
			return nil, err
		}
	}
	if int(must) < je.required {
		return nil, errors.New("one or more required fields missing")
	}
	return level0, nil
}

// 不处理json path中包含的点号问题
func (je *JsonPruneV2) jsonFlatten(m map[string]interface{}, prefix string) error {

	newmap := make(map[string]interface{})
	for key, value := range m {
		if value == nil {
			delete(m, key) // 删掉结果中的空指针, 不透传
			continue
		}
		switch v := value.(type) {
		case string:
			vstr := v
			parser, found := je.preParser[prefix+key]

			if found && parser != nil {
				lenParser := len(je.preParser[prefix+key])
				for index := 0; index < lenParser; index++ {
					vstrmap, err := parser[index].Parse(vstr)
					// 且geoip的提取失败时, 可以忽略
					if err != nil {
						if parser[index].Name() == text_parser.PresetGeoIP {
							continue
						}
						// 跳过解析失败的规则, 保留原字段
						// 下文所有递归调用的地方不用单独处理
						// 递归调用内部如果失败, 也是在这一行忽略
						if je.SkipExtractError {
							continue
						}
						// 配置了多个规则时, 前面的失败继续执行下一个, 直到最后一个失败
						if lenParser > 1 && index < lenParser-1 {
							continue
						}
						return errors.New("field[" + prefix + key + "] parser[" + parser[index].Name() + "] extract failed")
					}
					// 所有parser提取的结果都与原key同级, 而不是子结构
					// split方法添加前缀并展开
					if parser[index].Name() == text_parser.PresetSplit {
						for k, v := range vstrmap {
							newkey := key + "_" + text_parser.PresetSplit + "_" + k
							newmap[newkey] = v
						}
					} else if parser[index].Name() != text_parser.PresetJson {
						// json反序列化直接覆盖原结果, 不执行这一步
						// 如果存在同名, 则加一层前缀
						for k, v := range vstrmap {
							if _, found := m[k]; found {
								newmap[key+"_"+parser[index].Name()+"_"+k] = v
							} else {
								newmap[k] = v
							}
						}
					}
					// JSON反序列化直接覆盖
					// 非JSON类型根据配置决策, 是否直接覆盖原始字段
					// 默认不覆盖原始字段, 如果覆盖原始字段, 则下面不需要对newmap做额外处理
					if parser[index].Name() == text_parser.PresetJson || je.OverwriteOrigin {
						if je.OverwriteOrigin {
							newmap = make(map[string]interface{})
						}
						m[key] = vstrmap
						// 覆盖原始字段时, 需要对新的map结果执行一次preParser
						_, shouldRange := je.preParserPrefix[prefix+key]
						if shouldRange {
							if err := je.jsonFlatten(vstrmap, prefix+key+"."); err != nil {
								return err
							}
						}
					}
					if !je.MultiPreExtract {
						break
					}
				}
			}

		case map[string]interface{}:
			// 子map如果没有提取规则, 不再主动遍历
			_, shouldRange := je.preParserPrefix[prefix+key]
			if shouldRange {
				vmap := v
				if err := je.jsonFlatten(vmap, prefix+key+"."); err != nil {
					return err
				}
			}
		}
	}

	// 根据配置项决策是否覆盖原始字段
	// filter.PresetJson类型, 不会出现len(newmap) > 0的情况
	if !je.OverwriteOrigin {
		// 未覆盖原始字段时, 需要根据prefix前缀执行一次preParser
		// 覆盖原始字段时, 根据prefix+key执行一次preParser(见上文)
		if len(newmap) > 0 {
			if err := je.jsonFlatten(newmap, prefix); err != nil {
				return err
			}
		}
		for k, v := range newmap {
			m[k] = v
		}
	}
	return nil
}

func (je *JsonPruneV2) JsonPrune(origin map[string]interface{}, m map[string]interface{},
	prefix string, must *int32) error {

	rules, found := je.pruneIndex[prefix]
	if found {
		for i := range rules {
			// append只在原始枝干上扩展
			if rules[i].RuleType == LogExtractRuleTypeAppend {
				mvalue, err := rules[i].ExtractAppend()
				if err != nil {
					return err
				}
				if rules[i].Required {
					atomic.AddInt32(must, 1)
				}
				m[rules[i].targets[0]] = mvalue
			}
		}
	}
	for key, value := range m {
		if len(key) == 0 {
			delete(m, key) // 删除空的key
		}
		if value == nil {
			continue
		}

		// 是否有规则涉及到该枝干的处理规则, 如果没有, 则直接跳过该字段和可能存在的所有下层字段
		rules := je.pruneIndex[prefix+key]
		_, shouldRange := je.prunePrefixIndex[prefix+key]
		if !shouldRange && len(rules) == 0 {
			continue
		}

		switch v := value.(type) {
		case string, float64: // 暂时只处理这两种类型
			// 默认需要删除原key
			var deleteold bool
			for _, rule := range rules {
				if rule.RuleType == LogExtractRuleTypeSubMatch {
					if len(rule.targets) > 0 {
						deleteold = true
					}
				}
			}
			for _, rule := range rules {
				if rule.RuleType == LogExtractRuleTypeSubMatch {
					evalue, err := rule.ExtractSubMatch(v)
					if err != nil {
						return err
					}
					if evalue != nil {
						if rule.Required {
							atomic.AddInt32(must, 1)
						}
					}
					if len(rule.targets) == 0 {
						// 不需要嫁接, 直接替换原值
						// 如果有一条规则标识保留原key, 则不删除
						deleteold = false
						m[key] = evalue
					} else {
						addByPath(origin, rule.targets, evalue)
					}
				}
			}
			// 确认删除原key, 直接操作
			if deleteold {
				delete(m, key)
			}

		case map[string]interface{}:
			if shouldRange {
				if err := je.JsonPrune(origin, v, prefix+key+".", must); err != nil {
					return err
				}
			}
			// 所有的叶子已经被剪空, 主干不需要保留, 或者 原本就是空树枝, 也不再保留
			if len(v) == 0 {
				// 重新校验一下, 比如message.x被移动到 message, 结构已经发生了变化
				vv, found := m[key]
				if found {
					if vvv, ok := vv.(map[string]interface{}); ok && len(vvv) == 0 {
						delete(m, key)
					}
				}
			}
			// 其他类型直接透传, 暂时不处理
		}
		remove := false
		for _, rule := range rules {
			// 如果有嫁接操作, 在提取执行完成后再操作
			if rule.RuleType == LogExtractRuleTypeGraft {
				// 不需要操作的, required必须加一
				if rule.Required {
					atomic.AddInt32(must, 1)
				}
				if len(rule.targets) > 0 {
					delete(m, key)
					addByPath(origin, rule.targets, value)
				}
			} else if rule.RuleType == LogExtractRuleTypeRemove {
				remove = true
			}
		}
		// 如果有remove操作, 在所有的提取/嫁接执行完成后再操作
		if remove {
			delete(m, key)
		}
	}
	return nil
}

func (je *JsonPruneV2) addErrorTags(origin map[string]interface{}, err error) error {
	if err == nil {
		return errors.New("get FailInformation error")
	}
	errStr := err.Error()
	if errStr == "" {
		return errors.New("get FailInformation error")
	}
	origin["log_parse_failure"] = errStr
	return nil
}
