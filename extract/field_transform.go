package extract

import (
	"errors"
	"strconv"
	"strings"

	"github.com/flashcatcloud/fc-stash/map_iterator"
	"github.com/flashcatcloud/fc-stash/transform/deepcopy"
	"github.com/flashcatcloud/fc-stash/transform/field_format"
	"github.com/flashcatcloud/fc-stash/types"
)

var (
	ErrorOriginFieldNotFound       = errors.New("origin field not found")
	ErrorInternalErrorNilFormatter = errors.New("internal error, formatter is nil")
	ErrorInternalErrorNilMapping   = errors.New("internal error, mapping is nil")
)

func NewFieldTransform(settings FieldTransformSettings) (*FieldTransform, error) {
	if len(settings.OriginField) == 0 && settings.RuleType != FieldTransformTypeAppend {
		return nil, errors.New("invalid origin field, empty")
	}

	originPath := strings.Split(settings.OriginField, ".")
	var originField, targetField string
	var targetPath []string
	if len(settings.TargetField) > 0 && settings.TargetField != types.LogExtractJsonRoot {
		targetPath = strings.Split(settings.TargetField, ".")
	}
	if len(originPath) > 0 {
		originField = originPath[len(originPath)-1]
		originPath = originPath[:len(originPath)-1]
	}
	if len(targetPath) > 0 {
		targetField = targetPath[len(targetPath)-1]
		targetPath = targetPath[:len(targetPath)-1]
	}

	fieldTransform := &FieldTransform{
		settings:    settings,
		originPath:  originPath,
		originField: originField,
		targetPath:  targetPath,
		targetField: targetField,
	}
	if settings.RuleType == FieldTransformTypeFormat || settings.RuleType == FieldTransformTypeCloneAndFormat {
		formatter, err := field_format.NewFieldFormat(settings.FormatSettings, settings.TargetType)
		if err != nil {
			return nil, errors.New("field[" + settings.OriginField + "] invalid field format:" + err.Error())
		}
		fieldTransform.formatter = formatter
	}
	// 词表映射是一个特殊的规则
	if settings.AppendSettings.AppendType == FieldTransformAppendMethodLabelMapping {
		mapping, err := NewLabelMapping(settings.AppendSettings.LabelMapping)
		if err != nil {
			return nil, errors.New("field[" + settings.OriginField + "] invalid dict mapping:" + err.Error())
		}
		fieldTransform.mapping = mapping
	}
	return fieldTransform, nil
}

func GetLastMap(origin map[string]interface{}, path []string) (map[string]interface{}, bool) {
	// 只有一层, 代表origin就是目标
	if len(path) == 0 {
		return origin, true
	}
	lastvalue, ok := map_iterator.FindByPath(origin, path)
	if !ok {
		return nil, false
	}
	mapValue, ok := lastvalue.(map[string]interface{})
	if !ok {
		return nil, false
	}
	return mapValue, true
}

func (f *FieldTransform) Transform(origin map[string]interface{}) error {
	switch f.settings.RuleType {
	case FieldTransformTypeFormat:
		return f.Format(origin)

	case FieldTransformTypeAppend:
		return f.Append(origin)

	case FieldTransformTypeDelete:
		return f.Delete(origin)

	case FieldTransformTypeClone:
		return f.Clone(origin)

	case FieldTransformTypeRename:
		return f.Rename(origin)

	case FieldTransformTypeCloneAndFormat:
		return f.CloneAndFormat(origin)

	}
	return nil
}

// 格式化
func (f *FieldTransform) Format(origin map[string]interface{}) error {
	if f.formatter == nil {
		return ErrorInternalErrorNilFormatter
	}
	lastMap, ok := GetLastMap(origin, f.originPath)
	if !ok {
		if f.settings.DropIfNotFound {
			return ErrorOriginFieldNotFound
		}
		return nil
	}
	originValue, found := lastMap[f.originField]
	if !found || originValue == nil {
		if f.settings.DropIfNotFound {
			return ErrorOriginFieldNotFound
		}
		return nil
	}
	formatted, err := f.formatter.Format(originValue)
	if err != nil {
		return err
	}
	// 覆盖原值
	lastMap[f.originField] = formatted
	return nil
}

// 新增字段
func (f *FieldTransform) Append(origin map[string]interface{}) error {
	if f.settings.AppendSettings.AppendType == FieldTransformAppendMethodInput {
		lastMap, ok := GetLastMap(origin, f.targetPath)
		if !ok {
			// 目标字段不存在, 需要创建出来
			lastMap = make(map[string]interface{})
			err := map_iterator.SetByPath(origin, f.targetPath, lastMap)
			if err != nil {
				return err
			}
		}
		if _, found := lastMap[f.targetField]; found {
			if f.settings.IgnoreIfConflict {
				return nil
			}
		}
		var targetValue interface{}
		targetValue = f.settings.AppendSettings.InputValue
		if f.settings.TargetType == types.LogExtractValueTypeLong {
			targetValue, _ = strconv.ParseInt(f.settings.AppendSettings.InputValue, 10, 64)
		} else if f.settings.TargetType == types.LogExtractValueTypeFloat {
			targetValue, _ = strconv.ParseFloat(f.settings.AppendSettings.InputValue, 64)
		}
		lastMap[f.targetField] = targetValue
		return nil
	}
	if f.settings.AppendSettings.AppendType == FieldTransformAppendMethodLabelMapping {
		if f.mapping == nil {
			return ErrorInternalErrorNilMapping
		}
		parentMap, ok := f.mapping.GetSrcPath(origin)
		if !ok {
			return ErrorDictMappingParentFieldNotFound
		}
		values, err := f.mapping.GetValues(parentMap)
		if err != nil {
			return err
		}
		for key, value := range values {
			if _, found := parentMap[key]; found {
				if f.settings.IgnoreIfConflict {
					continue
				}
			}
			parentMap[key] = value
		}
		return nil
	}

	return nil
}

// 删除字段
func (f *FieldTransform) Delete(origin map[string]interface{}) error {
	lastMap, ok := GetLastMap(origin, f.originPath)
	// 删除的字段不存在, 忽略
	if !ok {
		return nil
	}
	delete(lastMap, f.originField)
	return nil
}

// 克隆字段
func (f *FieldTransform) Clone(origin map[string]interface{}) error {
	lastMap, ok := GetLastMap(origin, f.originPath)
	if !ok {
		if f.settings.DropIfNotFound {
			return ErrorOriginFieldNotFound
		}
		return nil
	}
	originValue, found := lastMap[f.originField]
	if !found {
		if f.settings.DropIfNotFound {
			return ErrorOriginFieldNotFound
		}
		return nil
	}
	// 目标字段值为nil, 忽略
	if originValue == nil {
		return nil
	}
	if ignore := f.checkIfIgnoreWhenConflict(origin, lastMap, originValue); ignore {
		return nil
	}

	// 使用 deepcopy 进行深拷贝
	// copier只有在明确知道类型的时候才能进行深拷贝
	copyvalue := deepcopy.Copy(originValue)
	// 如果目标字段是根节点, 则直接移动过去
	if f.settings.TargetField == types.LogExtractJsonRoot {
		f.moveToRoot(origin, copyvalue)
		return nil
	}
	// 使用拷贝后的值，并设置到正确的目标字段
	lastMap[f.targetField] = copyvalue
	return nil
}

// 重命名字段
func (f *FieldTransform) Rename(origin map[string]interface{}) error {
	lastMap, ok := GetLastMap(origin, f.originPath)
	if !ok {
		if f.settings.DropIfNotFound {
			return ErrorOriginFieldNotFound
		}
		return nil
	}
	originValue, found := lastMap[f.originField]
	if !found {
		if f.settings.DropIfNotFound {
			return ErrorOriginFieldNotFound
		}
		return nil
	}
	// 目标字段值为nil, 忽略
	if originValue == nil {
		return nil
	}
	// 判断是否存在冲突
	if ignore := f.checkIfIgnoreWhenConflict(origin, lastMap, originValue); ignore {
		return nil
	}

	// 如果目标字段是根节点, 则直接移动过去
	if f.settings.TargetField == types.LogExtractJsonRoot {
		delete(lastMap, f.originField)
		f.moveToRoot(origin, originValue)
		return nil
	}
	// 删除原字段, 并设置到目标字段
	delete(lastMap, f.originField)
	lastMap[f.targetField] = originValue
	return nil
}

// 克隆并格式化字段
func (f *FieldTransform) CloneAndFormat(origin map[string]interface{}) error {
	lastMap, ok := GetLastMap(origin, f.originPath)
	if !ok {
		if f.settings.DropIfNotFound {
			return ErrorOriginFieldNotFound
		}
		return nil
	}
	originValue, found := lastMap[f.originField]
	if !found {
		if f.settings.DropIfNotFound {
			return ErrorOriginFieldNotFound
		}
		return nil
	}
	// 目标字段值为nil, 忽略
	if originValue == nil {
		return nil
	}
	// 先执行格式化
	// 格式化要求originValue必须是字符串或数字
	formatted, err := f.formatter.Format(originValue)
	if err != nil {
		return err
	}
	// 再判断是否存在冲突
	if ignore := f.checkIfIgnoreWhenConflict(origin, lastMap, formatted); ignore {
		return nil
	}
	// 使用 deepcopy 进行深拷贝
	copyvalue := deepcopy.Copy(formatted)
	// 如果目标字段是根节点, 则直接执行移动过去
	if f.settings.TargetField == types.LogExtractJsonRoot {
		f.moveToRoot(origin, copyvalue)
		return nil
	}
	// 使用拷贝后的值，并设置到正确的目标字段
	lastMap[f.targetField] = copyvalue
	return nil
}

// 判断是否存在冲突
// 返回true代表不执行下一步动作, 直接退出
// targetField是__root__时, 代表将结果移动到origin的根节点
// 有两种情况:
// 1. 待移动的字段是map[string]interface{}, 则需要遍历map, 把每个元素都移动到origin的根节点
// 2. 待移动的字段是其他类型, 则直接移动到origin的根节点
func (f *FieldTransform) checkIfIgnoreWhenConflict(origin map[string]interface{},
	lastMap map[string]interface{}, value interface{}) bool {
	if f.settings.TargetField == types.LogExtractJsonRoot {
		if mapValue, ok := value.(map[string]interface{}); ok {
			// 任意map元素存在冲突, 则不执行下一步动作
			for key := range mapValue {
				if _, found := origin[key]; found {
					if f.settings.IgnoreIfConflict {
						return true
					}
				}
			}
		} else {
			if _, found := origin[f.originField]; found {
				if f.settings.IgnoreIfConflict {
					return true
				}
			}
		}
	} else {
		if _, found := lastMap[f.targetField]; found {
			if f.settings.IgnoreIfConflict {
				return true
			}
		}
	}
	return false
}

// .targetField 是 __root__时, 代表将结果移动到origin的根节点
// 同上
func (f *FieldTransform) moveToRoot(origin map[string]interface{}, value interface{}) {
	if mapValue, ok := value.(map[string]interface{}); ok {
		for key, value := range mapValue {
			origin[key] = value
		}
	} else {
		origin[f.originField] = value
	}
}
