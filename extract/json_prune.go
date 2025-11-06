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
	"github.com/flashcatcloud/fc-stash/utils"
)

// 剪枝模式, 与非剪枝模式同时支持, 兼容旧版
type JsonPrune struct {
	UUID             string                      `json:"-"`            // 用于配置更新
	IsCompiled       bool                        `json:"-"`            //
	PreFunction      []*filter.StringPreFunction `json:"pre_function"` // 前置处理
	PrefixMatch      []string                    `json:"prefix_match"` // 只支持第一层的过滤, 不做复杂的嵌套
	PreExtract       []*JsonPreSourceExtract     `json:"pre_extract"`  // 预提取
	Fields           []*FieldExtract             `json:"fields"`       // 提取规则
	pruneIndex       map[string][]*FieldExtract  `json:"-"`            // map索引
	prunePrefixIndex map[string]struct{}         `json:"-"`            // 如果不在这个列表中, jsonPrune()中可以不遍历
	required         int                         `json:"-"`            //
	prefixMatch      map[string]struct{}         `json:"-"`            // PrefixMatch的编译结果
	preParser        map[string]types.TextParser `json:"-"`            //
	preParserPrefix  map[string]struct{}         `json:"-"`            // 如果不在这个列表中, jsonFlatten中可以不遍历
}

func (le *JsonPrune) Init(ctx context.Context) error {
	return le.Compile()
}

func (le *JsonPrune) GetUUID() string {
	return le.UUID
}

func (le *JsonPrune) Compile(forceOrigin ...bool) error {
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
	// 用于决策, filter层是否需要返回 __origin__部分
	var preParserPrefix []string
	for _, field := range le.Fields {
		if field.RuleType != LogExtractRuleTypeAppend {
			if strings.HasSuffix(field.OriginField, "."+types.LogParserOriginText) {
				preParserPrefix = append(preParserPrefix,
					strings.TrimSuffix(field.OriginField, "."+types.LogParserOriginText))
			}
		}
	}
	if len(le.PreExtract) > 0 {
		le.preParser = make(map[string]types.TextParser)
		le.preParserPrefix = make(map[string]struct{})
		for _, pre := range le.PreExtract {
			// 旧版配置, 兼容一下, 直接丢掉, 不再处理
			if pre.Mode == filter.GlobalFuncStripAnsi {
				continue
			}
			needOrigin := false
			// forceOrigin是面向网页时使用
			if len(forceOrigin) > 0 && forceOrigin[0] {
				needOrigin = true
			} else {
				needOrigin = utils.StringArrayContains(preParserPrefix, pre.Field)
			}
			if err := pre.Compile(needOrigin); err != nil {
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
						if strings.HasPrefix(field.OriginField, pre.Field+".") {
							idx, err := strconv.Atoi(strings.SplitN(strings.TrimPrefix(field.OriginField, pre.Field+"."), ".", 2)[0])
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
			le.preParser[pre.Field] = pre.parser
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

func (le *JsonPrune) Handle(ctx context.Context, e *types.LogEvent) (types.ExtractedLog, error) {
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
func (je *JsonPrune) OriginJsonLogPrune(origin string) (ret map[string]interface{}, err error) {
	if !je.IsCompiled {
		return nil, errors.New("rules not compiled")
	}
	level0 := make(map[string]interface{})
	if err = json.Unmarshal([]byte(origin), &level0); err != nil {
		err = ErrorInvalidJsonString
		return
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
		return nil, err
	}
	must := int32(0)
	if len(je.pruneIndex) > 0 {
		if err := je.jsonPrune(level0, level0, "", &must); err != nil {
			return nil, err
		}
	}
	if int(must) < je.required {
		return nil, errors.New("one or more required fields missing")
	}
	return level0, nil
}

func (je *JsonPrune) OriginJsonLogPruneWithMap(level0 map[string]interface{}) (ret map[string]interface{}, err error) {
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
		return nil, err
	}
	must := int32(0)
	if len(je.pruneIndex) > 0 {
		if err := je.jsonPrune(level0, level0, "", &must); err != nil {
			return nil, err
		}
	}
	if int(must) < je.required {
		return nil, errors.New("one or more required fields missing")
	}
	return level0, nil
}

// 不处理json path中包含的点号问题
func (je *JsonPrune) jsonFlatten(m map[string]interface{}, prefix string) error {
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
				vstrmap, err := parser.Parse(vstr)
				// geoip的提取是特殊的, 提取结果与原key同级, 而不是子结构
				// 且geoip的提取失败时, 可以忽略
				if parser.Name() == text_parser.PresetGeoIP {
					if len(vstrmap) == 1 {
						for k, v := range vstrmap {
							m[k] = v
						}
					}
					continue
				}
				if err != nil {
					return errors.New("field[" + prefix + key + "] parser[" + parser.Name() + "] extract failed")
				}
				if len(vstrmap) > 0 {
					_, shouldRange := je.preParserPrefix[prefix+key]
					if shouldRange {
						if err := je.jsonFlatten(vstrmap, prefix+key+"."); err != nil {
							return err
						}
					}
					m[key] = vstrmap
				}
			}
			// 不再默认执行json extract, 必须显示指定
			// 如果默认执行json extract, 上层就没办法显示指定了
			/*
				else if len(prefix) == 0 && validKvPairJson(vstr) {
					// 只有第一层主动判断, 可以减少用户的配置成本
					// 如果显示的定义为codec:json, 可以进一步规避掉json.Valid()的消耗
					innermap := make(map[string]interface{})
					if err := json.Unmarshal([]byte(vstr), &innermap); err != nil {
						continue
					}
					_, shouldRange := je.preParserPrefix[prefix+key]
					if shouldRange {
						if err := je.jsonFlatten(innermap, prefix+key+"."); err != nil {
							return err
						}
					}
					m[key] = innermap
				}
			*/

		case map[string]interface{}:
			// 子map如果没有提取规则, 不再主动遍历
			_, shouldRange := je.preParserPrefix[prefix+key]
			if shouldRange {
				vmap := v
				if err := je.jsonFlatten(vmap, prefix+key+"."); err != nil {
					return err
				}
				m[key] = vmap
			}
		}
	}
	return nil
}

func (je *JsonPrune) jsonPrune(origin map[string]interface{}, m map[string]interface{},
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
				if err := je.jsonPrune(origin, v, prefix+key+".", must); err != nil {
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

// 如果目标节点不存在, 先忽略
func addByPath(origin map[string]interface{}, path []string, value interface{}) (err error) {
	if origin == nil {
		return
	}
	var lastmap interface{} = origin
	// 上移到根节点的特殊处理
	if len(path) == 1 && path[0] == types.LogExtractJsonRoot {
		switch vmap := value.(type) {
		case map[string]interface{}:
			for k, v := range vmap {
				origin[k] = v
			}
			return
		}
		// 如果value本身不是map结构, 则不能移动到根节点
		return
	}
	for i := 0; i < len(path)-1; i++ {
		if lastmap == nil {
			return
		}
		var next bool
		switch vmap := lastmap.(type) {
		case map[string]interface{}:
			lastmap, next = vmap[path[i]]
			if !next { // 不存在
				lastmap = make(map[string]interface{})
				vmap[path[i]] = lastmap
			}

		default: // lastmap的类型冲突, 直接返回
			return
		}
	}
	if lastmap == nil { // 不存在
		return
	}
	switch vmap := lastmap.(type) {
	case map[string]interface{}:
		vmap[path[len(path)-1]] = value
	}
	return
}
