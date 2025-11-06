package doris

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/flashcatcloud/fc-stash/types"
)

type StreamLoader struct {
	client *Client
}

// Write implements batch loading of logs into Doris using Stream Load API
func (sl *StreamLoader) Write(batch []types.ExtractedLog) error {
	if len(batch) == 0 {
		return nil
	}

	sl.client.mu.Lock()
	defer sl.client.mu.Unlock()

	// Convert batch data to JSON format for Stream Load
	jsonData, err := prepareJsonData(batch)
	if err != nil {
		return fmt.Errorf("prepare json data failed: %v", err)
	}

	// Create a unique label for this load
	label := fmt.Sprintf("fc_stash_%d", time.Now().UnixNano())

	// Construct Stream Load URL
	// Note: Assuming HTTP port is 8030 (default for FE HTTP port)

	url := fmt.Sprintf("http://%s/api/%s/%s/_stream_load",
		sl.client.config.FeHost, sl.client.config.Database, sl.client.config.Table)

	// Create HTTP request
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create stream load request failed: %v", err)
	}

	// Set request headers
	req.Header.Set("Expect", "100-continue")
	req.Header.Set("Content-Type", "application/json")
	// 用于指定 Doris 该次导入的标签，标签相同的数据无法多次导入。
	// 如果不指定 label，Doris 会自动生成一个标签。
	// 用户可以通过指定 label 的方式来避免一份数据重复导入的问题。
	// Doris 默认保留三天内的导入作业标签，可以 label_keep_max_second 调整保留时长。
	// 例如，指定本次导入 label 为 123，需要指定命令 -H "label:123"。
	// label 的使用，可以防止用户重复导入相同的数据。强烈推荐用户同一批次数据使用相同的 label。
	// 这样同一批次数据的重复请求只会被接受一次，保证了 At-Most-Once 当 label 对应的导入作业状态为 CANCELLED 时，该 label 可以再次被使用。
	req.Header.Set("label", label)
	req.Header.Set("format", "json")
	req.Header.Set("strip_outer_array", "true")
	req.SetBasicAuth(sl.client.config.Username, sl.client.config.Password)

	// Send request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("stream load request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read and parse response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response failed: %v", err)
	}

	// Parse response JSON
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parse response failed: %v, body: %s", err, string(body))
	}

	// Check if load was successful
	status, ok := result["Status"].(string)
	if !ok || (status != "Success" && status != "Publish Timeout") {
		errorMsg := "unknown error"
		if msg, ok := result["Message"].(string); ok {
			errorMsg = msg
		}
		return fmt.Errorf("stream load failed: %s, details: %s", status, errorMsg)
	}

	return nil
}

// prepareJsonData converts the batch of logs to JSON format suitable for Stream Load
func prepareJsonData(batch []types.ExtractedLog) ([]byte, error) {
	// For Stream Load with JSON format, we need an array of objects
	jsonData, err := json.Marshal(batch)
	if err != nil {
		return nil, fmt.Errorf("marshal json failed: %v", err)
	}

	return jsonData, nil
}
