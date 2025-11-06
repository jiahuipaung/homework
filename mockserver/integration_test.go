package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/flashcatcloud/fc-stash/sink/vmlogs"
	"github.com/flashcatcloud/fc-stash/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestVMLogsIntegration 完整的集成测试
// 测试流程：mockserver <- vmlogs output <- test data
func TestVMLogsIntegration(t *testing.T) {
	// 1. 启动 Mock Server
	config := &ServerConfig{
		Port:        19428,
		EnableAuth:  true,
		Username:    "test_user",
		Password:    "test_pass",
		EnableFault: false,
		Verbose:     true,
	}

	mockServer := NewMockServer(config)
	go func() {
		if err := mockServer.Start(); err != nil {
			t.Logf("Mock server stopped: %v", err)
		}
	}()
	defer mockServer.Stop()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)
	t.Log("✅ Mock server started on port 19428")

	// 2. 创建 VMLogs Output (使用项目的配置)
	output := &victorialogs.VMLogsOutput{
		Endpoints:          []string{fmt.Sprintf("http://localhost:%d", config.Port)},
		Username:           config.Username,
		Password:           config.Password,
		SkipTlsVerify:      true,
		Headers:            map[string]string{"X-Test": "integration"},
		RequestTimeout:     5 * time.Second,
		MaxRetryBackoff:    3 * time.Second,
		BatchActions:       10,                  // 每10条触发一次flush
		BatchSize:          1024,                // 1KB
		BatchFlushInterval: 2 * time.Second,     // 2秒触发一次flush
		MsgField:           "message",           // 指定message字段作为_msg_field
		TimeField:          "timestamp",         // 指定timestamp字段作为_time_field
		StreamFields:       []string{"service", "level"}, // 流字段
	}

	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	// 3. 初始化并启动 Output
	err := output.Init(ctx, 2)
	require.NoError(t, err, "Failed to init vmlogs output")
	t.Log("✅ VMLogs output initialized")

	err = output.Start(ctx, logger)
	require.NoError(t, err, "Failed to start vmlogs output")
	defer output.Stop(ctx)
	t.Log("✅ VMLogs output started")

	// 4. 发送测试数据
	t.Run("Send logs with different scenarios", func(t *testing.T) {
		// 场景1: 发送普通日志
		for i := 0; i < 5; i++ {
			log := types.ExtractedLog{
				"message":   fmt.Sprintf("Test log message %d", i),
				"timestamp": time.Now(),
				"service":   "test-service",
				"level":     "info",
				"user_id":   1000 + i,
				"request_id": fmt.Sprintf("req-%d", i),
			}
			err := output.Sink(ctx, log)
			assert.NoError(t, err, "Failed to sink log %d", i)
		}
		t.Log("✅ Sent 5 normal logs")

		// 场景2: 发送带有复杂数据的日志
		complexLog := types.ExtractedLog{
			"message":   "Complex log with nested data",
			"timestamp": time.Now(),
			"service":   "complex-service",
			"level":     "error",
			"metadata": map[string]interface{}{
				"error_code": 500,
				"details":    "Internal server error",
				"trace_id":   "abc-123-def",
			},
			"tags": []string{"production", "critical"},
		}
		err := output.Sink(ctx, complexLog)
		assert.NoError(t, err, "Failed to sink complex log")
		t.Log("✅ Sent complex log")

		// 场景3: 发送缺少stream字段的日志 (应该正常写入，只是会有warn日志)
		logWithoutStream := types.ExtractedLog{
			"message":   "Log without stream fields",
			"timestamp": time.Now(),
			// 缺少 service 和 level
		}
		err = output.Sink(ctx, logWithoutStream)
		assert.NoError(t, err, "Failed to sink log without stream fields")
		t.Log("✅ Sent log without stream fields")

		// 场景4: 触发批量flush (发送4条，总共10条，会触发flush)
		for i := 0; i < 4; i++ {
			log := types.ExtractedLog{
				"message":   fmt.Sprintf("Batch trigger log %d", i),
				"timestamp": time.Now(),
				"service":   "batch-service",
				"level":     "debug",
			}
			err := output.Sink(ctx, log)
			assert.NoError(t, err, "Failed to sink batch log %d", i)
		}
		t.Log("✅ Sent 4 more logs to trigger batch (total 10)")
	})

	// 5. 等待所有数据写入
	time.Sleep(3 * time.Second)
	t.Log("⏳ Waiting for all logs to be flushed...")

	// 6. 验证 Mock Server 收到的数据
	t.Run("Verify data received by mock server", func(t *testing.T) {
		mockServer.mu.RLock()
		totalLogs := len(mockServer.logs)
		mockServer.mu.RUnlock()

		t.Logf("📊 Mock server received %d logs", totalLogs)
		assert.GreaterOrEqual(t, totalLogs, 10, "Should receive at least 10 logs")

		// 检查日志内容
		mockServer.mu.RLock()
		defer mockServer.mu.RUnlock()

		foundNormalLog := false
		foundComplexLog := false
		foundLogWithoutStream := false

		for _, entry := range mockServer.logs {
			// 验证字段映射
			assert.NotEmpty(t, entry.Data, "Log data should not be empty")

			// 验证 query parameters
			if len(entry.StreamFields) > 0 {
				assert.Equal(t, []string{"service", "level"}, entry.StreamFields)
			}
			if entry.TimeField != "" {
				assert.Equal(t, "timestamp", entry.TimeField)
			}
			if entry.MsgField != "" {
				assert.Equal(t, "message", entry.MsgField)
			}

			// 检查特定日志
			if msg, ok := entry.Data["message"].(string); ok {
				if msg == "Test log message 0" {
					foundNormalLog = true
					assert.Equal(t, "test-service", entry.Data["service"])
					assert.Equal(t, "info", entry.Data["level"])
				}
				if msg == "Complex log with nested data" {
					foundComplexLog = true
					assert.Equal(t, "complex-service", entry.Data["service"])
					assert.NotNil(t, entry.Data["metadata"])
				}
				if msg == "Log without stream fields" {
					foundLogWithoutStream = true
				}
			}
		}

		assert.True(t, foundNormalLog, "Should find normal log")
		assert.True(t, foundComplexLog, "Should find complex log")
		assert.True(t, foundLogWithoutStream, "Should find log without stream fields")

		t.Log("✅ All logs verified successfully")
	})
}

// TestVMLogsWithAutoFields 测试自动字段生成
func TestVMLogsWithAutoFields(t *testing.T) {
	// 1. 启动 Mock Server
	config := &ServerConfig{
		Port:        19429,
		EnableAuth:  false,
		Verbose:     true,
	}

	mockServer := NewMockServer(config)
	go mockServer.Start()
	defer mockServer.Stop()
	time.Sleep(500 * time.Millisecond)

	// 2. 创建使用自动字段的 VMLogs Output
	output := &victorialogs.VMLogsOutput{
		Endpoints:          []string{fmt.Sprintf("http://localhost:%d", config.Port)},
		RequestTimeout:     5 * time.Second,
		BatchActions:       5,
		BatchFlushInterval: 1 * time.Second,
		MsgField:           "_msg", // 自动生成模式
		TimeField:          "",     // 空表示VMLogs自动生成
		StreamFields:       []string{}, // 不指定流字段
	}

	ctx := context.Background()
	logger := zap.NewNop()

	err := output.Init(ctx, 1)
	require.NoError(t, err)

	err = output.Start(ctx, logger)
	require.NoError(t, err)
	defer output.Stop(ctx)

	// 3. 发送测试数据
	for i := 0; i < 5; i++ {
		log := types.ExtractedLog{
			"log_content": fmt.Sprintf("Auto field test log %d", i),
			"index":       i,
		}
		err := output.Sink(ctx, log)
		assert.NoError(t, err)
	}

	time.Sleep(2 * time.Second)

	// 4. 验证
	mockServer.mu.RLock()
	defer mockServer.mu.RUnlock()

	assert.GreaterOrEqual(t, len(mockServer.logs), 5)
	t.Logf("✅ Received %d logs with auto fields", len(mockServer.logs))

	// 验证自动字段
	for _, entry := range mockServer.logs {
		// 应该包含自动生成的 _msg 字段
		if val, ok := entry.Data["_msg"]; ok {
			assert.Equal(t, victorialogs.DefaultMessage, val)
		}
	}
}

// TestVMLogsRetryAndFailover 测试重试和故障转移
func TestVMLogsRetryAndFailover(t *testing.T) {
	t.Skip("This test requires multiple mock servers")

	// 1. 启动两个 Mock Server
	server1Config := &ServerConfig{Port: 19430, Verbose: true}
	server2Config := &ServerConfig{Port: 19431, Verbose: true}

	server1 := NewMockServer(server1Config)
	server2 := NewMockServer(server2Config)

	go server1.Start()
	go server2.Start()
	defer server1.Stop()
	defer server2.Stop()

	time.Sleep(500 * time.Millisecond)

	// 2. 创建多节点的 VMLogs Output
	output := &victorialogs.VMLogsOutput{
		Endpoints: []string{
			fmt.Sprintf("http://localhost:%d", server1Config.Port),
			fmt.Sprintf("http://localhost:%d", server2Config.Port),
		},
		RequestTimeout:     5 * time.Second,
		MaxRetryBackoff:    1 * time.Second,
		BatchActions:       1,
		BatchFlushInterval: 1 * time.Second,
	}

	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	err := output.Init(ctx, 1)
	require.NoError(t, err)

	err = output.Start(ctx, logger)
	require.NoError(t, err)
	defer output.Stop(ctx)

	// 3. 模拟第一个节点故障
	server1.healthStatus = 1 // 设置为不健康

	// 4. 发送数据（应该自动转移到第二个节点）
	for i := 0; i < 10; i++ {
		log := types.ExtractedLog{
			"message": fmt.Sprintf("Failover test log %d", i),
			"index":   i,
		}
		err := output.Sink(ctx, log)
		assert.NoError(t, err)
	}

	time.Sleep(3 * time.Second)

	// 5. 验证负载分布
	server1.mu.RLock()
	count1 := len(server1.logs)
	server1.mu.RUnlock()

	server2.mu.RLock()
	count2 := len(server2.logs)
	server2.mu.RUnlock()

	t.Logf("Server1 received: %d logs (unhealthy)", count1)
	t.Logf("Server2 received: %d logs", count2)

	// server1 不健康，应该主要是 server2 收到数据
	assert.GreaterOrEqual(t, count2, 5, "Server2 should receive most logs")
}

// TestVMLogsBenchmark 性能测试
func TestVMLogsBenchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping benchmark in short mode")
	}

	// 1. 启动 Mock Server
	config := &ServerConfig{
		Port:    19432,
		Verbose: false, // 关闭详细日志以提高性能
	}

	mockServer := NewMockServer(config)
	go mockServer.Start()
	defer mockServer.Stop()
	time.Sleep(500 * time.Millisecond)

	// 2. 创建高性能配置的 VMLogs Output
	output := &victorialogs.VMLogsOutput{
		Endpoints:          []string{fmt.Sprintf("http://localhost:%d", config.Port)},
		RequestTimeout:     10 * time.Second,
		BatchActions:       1000,             // 大批次
		BatchSize:          5 * 1024 * 1024,  // 5MB
		BatchFlushInterval: 5 * time.Second,  // 较长的flush间隔
	}

	ctx := context.Background()
	logger := zap.NewNop()

	err := output.Init(ctx, 4) // 4个worker
	require.NoError(t, err)

	err = output.Start(ctx, logger)
	require.NoError(t, err)
	defer output.Stop(ctx)

	// 3. 发送大量数据
	totalLogs := 10000
	startTime := time.Now()

	for i := 0; i < totalLogs; i++ {
		log := types.ExtractedLog{
			"message":   fmt.Sprintf("Benchmark log %d", i),
			"timestamp": time.Now(),
			"service":   "benchmark-service",
			"index":     i,
		}
		err := output.Sink(ctx, log)
		if err != nil {
			t.Logf("Error sinking log %d: %v", i, err)
		}
	}

	// 等待所有数据刷新
	time.Sleep(10 * time.Second)

	duration := time.Since(startTime)
	logsPerSecond := float64(totalLogs) / duration.Seconds()

	mockServer.mu.RLock()
	receivedLogs := len(mockServer.logs)
	mockServer.mu.RUnlock()

	t.Logf("📊 Performance Results:")
	t.Logf("   Total logs sent: %d", totalLogs)
	t.Logf("   Logs received: %d", receivedLogs)
	t.Logf("   Duration: %v", duration)
	t.Logf("   Throughput: %.2f logs/sec", logsPerSecond)
	t.Logf("   Loss rate: %.2f%%", float64(totalLogs-receivedLogs)/float64(totalLogs)*100)

	// 验证大部分日志都收到了（允许少量丢失）
	assert.GreaterOrEqual(t, receivedLogs, int(float64(totalLogs)*0.95),
		"Should receive at least 95%% of logs")
}

// 辅助函数：打印日志详情
func printLogDetails(t *testing.T, logs []LogEntry) {
	t.Log("📋 Detailed log entries:")
	for i, entry := range logs {
		data, _ := json.MarshalIndent(entry.Data, "  ", "  ")
		t.Logf("  [%d] Timestamp: %v", i, entry.Timestamp)
		t.Logf("  [%d] Data: %s", i, string(data))
		if len(entry.StreamFields) > 0 {
			t.Logf("  [%d] StreamFields: %v", i, entry.StreamFields)
		}
	}
}
