package extract

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/flashcatcloud/fc-stash/types"
)

var (
	ErrorDictMappingSchemaNotFound      = errors.New("dict mapping schema not found")
	ErrorDictMappingParentFieldEmpty    = errors.New("dict mapping parent field empty")
	ErrorDictMappingParentFieldNotFound = errors.New("dict mapping parent not found")
	ErrorDictMappingSrcsNotMatch        = errors.New("dict mapping srcs not match")
)

var (
	dictMappingSchemas atomic.Value
)

// 字典映射
type LabelMappingSettings struct {
	SchemaID    int64    `json:"schema_id,omitempty"`    // 夜莺的词表ID
	ParentField string   `json:"parent_field,omitempty"` // 父字段
	Srcs        []string `json:"srcs,omitempty"`         // 源字段, 主要用于前端展示, 与n9e schema中的source keys一致
	Dsts        []string `json:"dsts,omitempty"`
}

type N9eLabelMappingSchema struct {
	Mapping map[string]map[string]string // src kvs -> dst k -> dst v
}

type LabelMapping struct {
	settings   LabelMappingSettings
	parentPath []string
	sortedSrcs []string
}

func init() {
	dictMappingSchemas.Store(make(map[int64]*N9eLabelMappingSchema))
}

// 上游从n9e-plus获取配置时, 一并初始化
func SetLabelMappingSchemas(schemas map[int64]map[string]map[string]string) {
	schemaMap := make(map[int64]*N9eLabelMappingSchema)
	for k, v := range schemas {
		schemaMap[k] = &N9eLabelMappingSchema{
			Mapping: v,
		}
	}
	dictMappingSchemas.Store(schemaMap)
}

func NewLabelMapping(settings LabelMappingSettings) (*LabelMapping, error) {
	if settings.SchemaID == 0 {
		return nil, ErrorDictMappingSchemaNotFound
	}
	if len(settings.ParentField) == 0 {
		return nil, ErrorDictMappingParentFieldEmpty
	}
	ret := &LabelMapping{
		settings: settings,
	}
	ret.sortedSrcs = append(ret.sortedSrcs, settings.Srcs...)
	sort.Strings(ret.sortedSrcs)
	// 如果父字段是根节点，则直接返回
	if settings.ParentField != types.LogExtractJsonRoot {
		ret.parentPath = strings.Split(settings.ParentField, ".")
	}
	return ret, nil
}

func (d *LabelMapping) GetSrcPath(origin map[string]interface{}) (map[string]interface{}, bool) {
	lastMap, ok := GetLastMap(origin, d.parentPath)
	if !ok {
		return nil, false
	}
	return lastMap, true
}

// 获取映射后的值
// 忽略所有错误, 避免影响日志落库
func (d *LabelMapping) GetValues(parentMap map[string]interface{}) (map[string]string, error) {
	schema, ok := dictMappingSchemas.Load().(map[int64]*N9eLabelMappingSchema)[d.settings.SchemaID]
	if !ok {
		return nil, ErrorDictMappingSchemaNotFound
	}

	kvs := make(map[string]string)
	// 如果有不存在的key, 代表键值不全, 则返回value为空的结果
	for _, dstK := range d.settings.Dsts {
		kvs[dstK] = ""
	}
	if schema == nil {
		return kvs, nil
	}
	srcKvs := make([]string, 0, len(d.sortedSrcs))
	notAllExist := false
	for _, k := range d.sortedSrcs {
		v, ok := parentMap[k]
		if !ok {
			notAllExist = true
			break
		}
		srcKvs = append(srcKvs, fmt.Sprintf("%s=%s", k, v))
	}
	if notAllExist {
		// 如果键值不全, 则返回空结果
		return kvs, nil
	}
	srcKvsStr := strings.Join(srcKvs, ",")
	dstKvs, ok := schema.Mapping[srcKvsStr]
	if !ok {
		// 如果映射不存在, 则返回空结果
		return kvs, nil
	}
	for _, dstK := range d.settings.Dsts {
		if dstValue, ok := dstKvs[dstK]; ok && len(dstValue) > 0 {
			kvs[dstK] = dstValue
		}
	}
	return kvs, nil
}
