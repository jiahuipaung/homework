package extract

import (
	"github.com/flashcatcloud/fc-stash/transform/field_format"
	"github.com/flashcatcloud/fc-stash/transform/filter"
)

const (
	FieldTransformTypeFormat         = "format"
	FieldTransformTypeCloneAndFormat = "clone_and_format"
	FieldTransformTypeAppend         = "append"
	FieldTransformTypeDelete         = "delete"
	FieldTransformTypeRename         = "rename" // LogExtractJsonRoot 代表移动到根节点
	FieldTransformTypeClone          = "clone"
)

const (
	FieldTransformAppendMethodInput        = "input"         // 输入字段
	FieldTransformAppendMethodLabelMapping = "label_mapping" // 字典映射
)

// 日志提取的关键配置, 存储在数据库中, 由fc-stash执行
// JsonPruneV2的父集, 可以替换JsonPruneV2
type TransformSettings struct {
	PreFunction      []*filter.StringPreFunction `json:"pre_function"`       // 筛选和预处理
	PrefixMatch      []string                    `json:"prefix_match"`       // 字段提取(第一层的过滤)
	PreExtract       []*JsonPreSourceExtract     `json:"pre_extract"`        // 复杂文本提取
	MultiPreExtract  bool                        `json:"multi_pre_extract"`  // 复杂文本提取的标记: 单字段执执行多次, 如果有
	SkipExtractError bool                        `json:"skip_extract_error"` // 复杂文本提取的标记: 跳过单字段多规则中的解析失败的规则
	OverwriteOrigin  bool                        `json:"overwrite_origin"`   // 复杂文本提取的标记: 是否要覆盖原字段, json直接覆盖
	FieldsTransform  []FieldTransformSettings    `json:"fields_transform"`   // 字段转换, 按顺序执行
	// Deprecated
	Fields []*FieldExtract `json:"fields,omitempty"` // 提取规则
}

// 单个字段的转换
// 支持格式化、克隆并格式化、新增、删除、复制、重命名
type FieldTransformSettings struct {
	RuleType         string                           `json:"rule_type"`
	OriginField      string                           `json:"origin_field,omitempty"`
	OriginType       string                           `json:"origin_type,omitempty"`
	TargetField      string                           `json:"target_field,omitempty"`
	TargetType       string                           `json:"target_type,omitempty"`
	FormatSettings   field_format.FieldFormatSettings `json:"format_settings"`    // rule_type为format或clone_and_format时, 格式化参数
	AppendSettings   AppendSettings                   `json:"append_settings"`    // rule_type为append时, 追加参数
	DropIfNotFound   bool                             `json:"drop_if_not_found"`  // rule_type为format/clone_and_format/rename时有效
	IgnoreIfConflict bool                             `json:"ignore_if_conflict"` // rule_type为append/clone/clone_and_format/rename时有效
}

type AppendSettings struct {
	AppendType   string               `json:"append_type,omitempty"` // 追加类型
	InputValue   string               `json:"input_value,omitempty"` // 输入值
	LabelMapping LabelMappingSettings `json:"label_mapping"`         // 字典映射
}
