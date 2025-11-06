package map_iterator

import "errors"

// 设置值到指定路径, 如果路径不存在, 需要递归的创建出来
func SetByPath(origin map[string]interface{}, path []string, value interface{}) error {
	if origin == nil {
		return errors.New("origin map cannot be nil")
	}

	if len(path) == 0 {
		return errors.New("path cannot be empty")
	}

	var current map[string]interface{} = origin

	// 遍历到倒数第二个路径元素
	for i := 0; i < len(path)-1; i++ {
		key := path[i]

		// 检查当前层级是否存在
		if nextMap, exists := current[key]; exists {
			if nextMapTyped, ok := nextMap.(map[string]interface{}); ok {
				current = nextMapTyped
				continue
			}
			// 如果存在但不是 map，返回错误
			return errors.New("path element is not a map")
		}

		// 创建新的 map 并设置到当前层级
		newMap := make(map[string]interface{})
		current[key] = newMap
		current = newMap
	}

	// 设置最终值
	current[path[len(path)-1]] = value
	return nil
}
