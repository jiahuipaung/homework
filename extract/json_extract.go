package extract

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/flashcatcloud/fc-stash/transform/filter"
	"github.com/flashcatcloud/fc-stash/types"
)

// 旧的逻辑, 需要保留, 兼容旧的配置
// 非剪枝模式, 只提取已定义的目标节点
type JsonExtract struct {
	UUID             string                      `json:"-"`            // 用于配置更新
	IsCompiled       bool                        `json:"-"`            //
	PreFunction      []*filter.StringPreFunction `json:"pre_function"` // 前置处理
	PrefixMatch      []string                    `json:"prefix_match"` // 需要提取的字段, 是 Fields 的前缀; 如果为空代表全匹配
	PreExtract       []*JsonPreSourceExtract     `json:"pre_extract"`  // 预提取
	Fields           []*FieldExtract             `json:"fields"`       // 提取规则
	RemainRawMessage bool                        `json:"-"`            // 是否保留原始消息, 由主题中的配置项决定
	matchIndex       map[string][]*FieldExtract  `json:"-"`            // map索引
	matchFieldSplit  map[string][]string         `json:"-"`            // origin_field前置处理
	appendIndex      []*FieldExtract             `json:"-"`            // 按顺序遍历即可
	requiredIndex    map[string]bool             `json:"-"`            //
	prefixMatch      map[int]map[string]struct{} `json:"-"`            // PrefixMatch的编译结果
	preParser        map[string]types.TextParser `json:"-"`            //
}

func (le *JsonExtract) Init(ctx context.Context) error {
	return le.Compile()
}

func (le *JsonExtract) GetUUID() string {
	return le.UUID
}

func (le *JsonExtract) Compile() error {
	// 避免重复执行
	if le.IsCompiled {
		return nil
	}
	if len(le.PreExtract) > 0 {
		le.preParser = make(map[string]types.TextParser)
		for _, pre := range le.PreExtract {
			// ansi remove不是字段操作, 而是全局操作
			if pre.Mode == filter.GlobalFuncStripAnsi {
				continue
			}
			// 默认返回__origin__字段, 兼容之前的逻辑
			if err := pre.Compile(true); err != nil {
				return err
			}
			le.preParser[pre.Field] = pre.parser
		}
	}
	if len(le.PrefixMatch) > 0 {
		le.prefixMatch = make(map[int]map[string]struct{})
		for _, filter := range le.PrefixMatch {
			if len(filter) == 0 {
				continue
			}
			for level, split := range strings.Split(filter, ".") {
				if len(split) == 0 {
					continue
				}
				if _, found := le.prefixMatch[level]; !found {
					le.prefixMatch[level] = make(map[string]struct{})
				}
				le.prefixMatch[level][split] = struct{}{}
			}
		}
	}
	if len(le.Fields) == 0 {
		return ErrorLogExtractEmptyFields
	}
	for _, field := range le.Fields {
		if err := field.Compile(); err != nil {
			return err
		}
	}
	le.matchIndex = make(map[string][]*FieldExtract)
	le.matchFieldSplit = make(map[string][]string)
	le.requiredIndex = make(map[string]bool)
	for _, field := range le.Fields {
		if field.SystemRequired || field.Required {
			le.requiredIndex[field.Key] = true
		}
		if field.RuleType != LogExtractRuleTypeAppend {
			if _, found := le.matchIndex[field.OriginField]; !found {
				le.matchIndex[field.OriginField] = make([]*FieldExtract, 0)
			}
			le.matchIndex[field.OriginField] = append(le.matchIndex[field.OriginField], field)

			if _, found := le.matchFieldSplit[field.OriginField]; !found {
				// 只需要执行一次
				le.matchFieldSplit[field.OriginField] = strings.Split(field.OriginField, ".")
			}
		} else {
			le.appendIndex = append(le.appendIndex, field)
		}
	}

	le.IsCompiled = true
	return nil
}

func (le *JsonExtract) Handle(ctx context.Context, e *types.LogEvent) (types.ExtractedLog, error) {
	if e == nil || e.Message == nil {
		return nil, errors.New("nil event or empty message")
	}
	// 一旦某一步执行失败, 则直接退出
	extracted, _, err := le.Extract(e, true)
	if err != nil {
		return nil, err
	}
	// disable
	if len(extracted) == 0 {
		return nil, nil
	}
	if le.RemainRawMessage {
		extracted[types.LogEventMessageKey] = e.Message
	}
	return extracted, nil
}

// 启动优化相关工作
func (le *JsonExtract) Extract(e *types.LogEvent, breakWheneverFail bool) (
	map[string]interface{}, map[string]error, error) {
	if !le.IsCompiled {
		return nil, nil, errors.New("rules not compiled")
	}
	message, ok := e.Message.(string)
	if !ok || len(message) == 0 {
		return nil, nil, errors.New("nil event or empty message")
	}

	vmap, err := le.OriginJsonLogFlatten(message)
	if err != nil {
		return nil, nil, err
	}
	var size int
	for _, rules := range le.matchIndex {
		size += len(rules)
	}
	size += len(le.appendIndex)
	ret := make(map[string]interface{})
	var retErr map[string]error
	must := 0
	for originField, rules := range le.matchIndex {
		if len(originField) == 0 || len(rules) == 0 {
			continue
		}
		depths, found := le.matchFieldSplit[originField]
		if !found { // fatal error
			continue
		}

		var lastmap interface{} = vmap
		var valueMatch bool = true
		for i := 0; i < len(depths); i++ {
			if lastmap == nil {
				valueMatch = false
				break
			}
			if reflect.TypeOf(lastmap).Kind() != reflect.Map {
				valueMatch = false
				break
			}
			var next bool
			lastmap, next = lastmap.(map[string]interface{})[depths[i]]
			if !next {
				valueMatch = false
				break
			}
		}
		if !valueMatch {
			continue
		}

		for _, rule := range rules {
			value, err := rule.ExtractSubMatch(lastmap)
			if err != nil {
				// 实际执行过程中, 如果设置了空结果丢弃, 那么会在这一步直接退出, ExtractSubMatch()内部做了判断
				// 空值不认为是错误, 除非用户设置了必须字段
				if breakWheneverFail && err != ErrorJsonExtractValueTypeNil {
					return nil, nil, fmt.Errorf("field[%s] extract failed:%v", originField, err)
				}
				// 使用时初始化
				if len(retErr) == 0 {
					retErr = make(map[string]error)
				}
				retErr[rule.Key] = err
			}
			// stash的正常流程中会出现value==nil的情况
			if value != nil {
				ret[rule.Key] = value
			}
			if _, found := le.requiredIndex[rule.Key]; found {
				if value != nil {
					must += 1
				}
			}
		}
	}
	for _, rule := range le.appendIndex {
		value, err := rule.ExtractAppend()
		if err != nil {
			if breakWheneverFail {
				return nil, nil, err
			}
			// 使用时初始化
			if len(retErr) == 0 {
				retErr = make(map[string]error)
			}
			retErr[rule.Key] = err
		}
		// 加一下安全保护, 避免nil指针报错
		if value != nil {
			ret[rule.Key] = value
		}
		if _, found := le.requiredIndex[rule.Key]; found {
			// 规则同上
			if value != nil {
				must += 1
			}
		}
	}
	if must < len(le.requiredIndex) {
		return nil, nil, errors.New("one or more required fields missing")
	}
	return ret, retErr, nil
}

// 简化OriginJsonLogFlatten()逻辑, 非必须的字段直接过滤
func (je *JsonExtract) OriginJsonLogFlatten(origin string) (ret map[string]interface{}, err error) {
	// 取消该步骤, 进一步提高性能, !json.Valid(bs)

	level0 := make(map[string]interface{})
	// json Unmarshal会丢掉ansi的特征, 提前抹除掉
	removeAnsi := false
	for i := range je.PreExtract {
		if je.PreExtract[i].Mode == filter.GlobalFuncStripAnsi {
			removeAnsi = true
			break
		}
	}
	if removeAnsi {
		origin = filter.RemoveAnsiEscape(origin)
	}
	if err = json.Unmarshal([]byte(origin), &level0); err != nil {
		err = ErrorInvalidJsonString
		return
	}
	if err := je.jsonFlatten(level0, ""); err != nil {
		return nil, err
	}
	return level0, nil
}

func (je *JsonExtract) hasNestedPreParser(target string) bool {
	for key := range je.preParser {
		if strings.HasPrefix(key, target+".") {
			return true
		}
	}
	return false
}

func (je *JsonExtract) nestedPreParser(m map[string]interface{}, prefix string) error {
	for key, value := range m {
		switch reflect.TypeOf(value).Kind() {
		case reflect.String:
			vstr := value.(string)
			parser, found := je.preParser[prefix+key]
			if found && parser != nil {
				vstrmap, err := parser.Parse(vstr)
				if err != nil {
					return errors.New("field[" + prefix + key + "] parser[" + parser.Name() + "] extract failed")
				}
				if len(vstrmap) > 0 {
					m[key] = vstrmap
				}
			}
		}
	}
	return nil
}

func (je *JsonExtract) jsonFlatten(m map[string]interface{}, prefix string) error {
	var filters map[string]struct{}
	if len(je.prefixMatch) > 0 {
		filters = je.prefixMatch[strings.Count(prefix, ".")]
	}

	for key, value := range m {
		if value == nil {
			continue
		}
		// 非提取字段不处理, 从原始map中删掉
		if _, found := filters[key]; !found && len(filters) > 0 { // len(filters) == 0 代表该层全匹配
			delete(m, key)
			continue
		}
		resetValue := false
		if strings.Contains(key, ".") {
			// 统一替换key中的 ".", 该符号认为是json struct的分层标识
			delete(m, key)
			key = strings.Replace(key, ".", "_", -1)
			resetValue = true
		}

		switch reflect.TypeOf(value).Kind() {
		case reflect.Float64:
			if resetValue {
				m[key] = value
			}

		case reflect.String:
			vstr := value.(string)
			parser, found := je.preParser[prefix+key]
			if found && parser != nil {
				vstrmap, err := parser.Parse(vstr)
				if err != nil {
					return errors.New("field[" + prefix + key + "] parser[" + parser.Name() + "] extract failed")
				}
				if len(vstrmap) > 0 {
					if je.hasNestedPreParser(prefix + key) {
						if err := je.nestedPreParser(vstrmap, prefix+key+"."); err != nil {
							return err
						}
					}
					m[key] = vstrmap
				}
			} else if validKvPairJson(vstr) {
				// TODO: 如果显示的定义为codec:json, 可以进一步规避掉json.Valid()的消耗
				innermap := make(map[string]interface{})
				if err := json.Unmarshal([]byte(vstr), &innermap); err != nil {
					continue
				}
				je.jsonFlatten(innermap, key+".")
				m[key] = innermap

			} else if resetValue { // 保持原值
				m[key] = value
			}

		case reflect.Map:
			vmap := value.(map[string]interface{})
			je.jsonFlatten(vmap, key+".")
			m[key] = vmap
		}
	}
	return nil
}

func validKvPairJson(v string) bool {
	return strings.Contains(v, "{") && json.Valid([]byte(v))
}

// 面向网页的解析
func OriginJsonLogFlatten2(origin string,
	prefixMatch []string,
	preExtract []*JsonPreSourceExtract) (fields []string, values []interface{}, types []reflect.Kind, err error) {
	je := new(JsonExtract)
	je.PrefixMatch = prefixMatch
	je.PreExtract = preExtract
	if err = je.Compile(); err != nil && err != ErrorLogExtractEmptyFields {
		return
	}
	var message map[string]interface{}
	message, err = je.OriginJsonLogFlatten(origin)
	if err != nil {
		return
	}
	fields, values, types = JsonFlattenFromMap(message)
	return
}
