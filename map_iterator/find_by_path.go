package map_iterator

import "strings"

func FindByField(origin map[string]interface{}, field string) (interface{}, bool) {
	if len(origin) == 0 || len(field) == 0 {
		return nil, false
	}

	return FindByPath(origin, strings.Split(field, "."))
}

func FindByPath(origin map[string]interface{}, path []string) (interface{}, bool) {
	if len(origin) == 0 {
		return nil, false
	}
	// 如果path为空, 则返回origin
	if len(path) == 0 {
		return origin, true
	}

	var lastmap interface{} = origin
	for i := 0; i < len(path)-1; i++ {
		if lastmap == nil {
			return nil, false
		}
		var next bool
		switch vmap := lastmap.(type) {
		case map[string]interface{}:
			lastmap, next = vmap[path[i]]
			if !next {
				return nil, false
			}

		default:
			return nil, false
		}
	}
	if lastmap == nil {
		return nil, false
	}
	switch vmap := lastmap.(type) {
	case map[string]interface{}:
		v, found := vmap[path[len(path)-1]]
		return v, found
	}

	return nil, false
}
