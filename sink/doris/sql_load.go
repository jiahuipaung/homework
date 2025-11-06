package doris

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flashcatcloud/fc-stash/types"
)

type SqlLoader struct {
	client *Client
}

// Load 使用sql的方式插入数据
// 对于日志这种量大的场景，不推荐使用
func (sl *SqlLoader) Write(batch []types.ExtractedLog) error {
	if len(batch) == 0 {
		return nil
	}

	sl.client.mu.Lock()
	defer sl.client.mu.Unlock()

	// 构建批量插入SQL
	var (
		columns []string
		values  []string
		args    []interface{}
	)

	// 从第一条数据获取列名
	for k := range batch[0] {
		columns = append(columns, k)
	}

	// 构建values部分
	for _, data := range batch {
		placeholders := make([]string, len(columns))
		for i, col := range columns {
			placeholders[i] = "?"
			var arg interface{}
			switch data[col].(type) {
			case map[string]interface{}:
				arg, _ = json.Marshal(data[col])
			default:
				arg = data[col]
			}
			args = append(args, arg)
		}
		values = append(values, "("+strings.Join(placeholders, ",")+")")
	}

	// 构建完整SQL
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s",
		sl.client.config.Table,
		strings.Join(columns, ","),
		strings.Join(values, ","),
	)

	// 执行插入
	_, err := sl.client.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("insert into doris %s.%s failed: %w", sl.client.config.Database, sl.client.config.Table, err)
	}

	return nil
}
