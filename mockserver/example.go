package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/flashcatcloud/fc-stash/sink/vmlogs"
	"github.com/flashcatcloud/fc-stash/types"
	"go.uber.org/zap"
)

// Example: 一个简单的集成测试示例
// 演示如何使用 mockserver 测试 vmlogs 模块
func main() {
	fmt.Println("🚀 VMLogs Integration Test Example")
	fmt.Println("==================================")

	// 1. 启动 Mock Server
	fmt.Println("\n📌 Step 1: Starting mock server...")
	config := &ServerConfig{
		Port:        9428,
		EnableAuth:  true,
		Username:    "root",
		Password:    "root.2020",
		EnableFault: false,
		Verbose:     true,
	}

	mockServer := NewMockServer(config)
	go func() {
		if err := mockServer.Start(); err != nil {
			log.Printf("Mock server stopped: %v", err)
		}
	}()
	defer mockServer.Stop()

	// Wait for server to start
	time.Sleep(1 * time.Second)
	fmt.Printf("✅ Mock server started at http://localhost:%d\n", config.Port)

	// 2. 创建 VMLogs Output
	fmt.Println("\n📌 Step 2: Creating VMLogs output...")
	output := &victorialogs.VMLogsOutput{
		Endpoints:          []string{fmt.Sprintf("http://localhost:%d", config.Port)},
		Username:           config.Username,
		Password:           config.Password,
		SkipTlsVerify:      true,
		RequestTimeout:     5 * time.Second,
		MaxRetryBackoff:    3 * time.Second,
		BatchActions:       5,                   // 每5条flush一次
		BatchSize:          1024,                // 1KB
		BatchFlushInterval: 2 * time.Second,     // 2秒flush一次
		MsgField:           "message",           // message字段作为_msg
		TimeField:          "timestamp",         // timestamp字段作为_time
		StreamFields:       []string{"service", "level"}, // 流字段
	}

	ctx := context.Background()
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}

	// 3. 初始化并启动 Output
	fmt.Println("\n📌 Step 3: Initializing VMLogs output...")
	if err := output.Init(ctx, 2); err != nil {
		log.Fatalf("Failed to init output: %v", err)
	}

	if err := output.Start(ctx, logger); err != nil {
		log.Fatalf("Failed to start output: %v", err)
	}
	defer output.Stop(ctx)
	fmt.Println("✅ VMLogs output initialized and started")

	// 4. 发送测试数据
	fmt.Println("\n📌 Step 4: Sending test logs...")

	// 场景1: 发送普通日志
	fmt.Println("\n  📝 Scenario 1: Normal logs")
	for i := 0; i < 3; i++ {
		log := types.ExtractedLog{
			"message":    fmt.Sprintf("Test log message %d", i),
			"timestamp":  time.Now(),
			"service":    "test-service",
			"level":      "info",
			"user_id":    1000 + i,
			"request_id": fmt.Sprintf("req-%d", i),
		}
		if err := output.Sink(ctx, log); err != nil {
			fmt.Printf("  ❌ Failed to send log %d: %v\n", i, err)
		} else {
			fmt.Printf("  ✅ Sent log %d\n", i)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 场景2: 发送带有复杂数据的日志
	fmt.Println("\n  📝 Scenario 2: Complex log with nested data")
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
	if err := output.Sink(ctx, complexLog); err != nil {
		fmt.Printf("  ❌ Failed to send complex log: %v\n", err)
	} else {
		fmt.Println("  ✅ Sent complex log")
	}

	// 场景3: 触发批量flush (再发送1条，总共5条)
	fmt.Println("\n  📝 Scenario 3: Trigger batch flush")
	batchLog := types.ExtractedLog{
		"message":   "This should trigger batch flush",
		"timestamp": time.Now(),
		"service":   "batch-service",
		"level":     "debug",
	}
	if err := output.Sink(ctx, batchLog); err != nil {
		fmt.Printf("  ❌ Failed to send batch log: %v\n", err)
	} else {
		fmt.Println("  ✅ Sent log to trigger batch (total 5 logs)")
	}

	// 5. 等待数据flush
	fmt.Println("\n📌 Step 5: Waiting for flush...")
	time.Sleep(3 * time.Second)

	// 6. 查看统计信息
	fmt.Println("\n📌 Step 6: Checking statistics...")
	mockServer.mu.RLock()
	totalLogs := len(mockServer.logs)
	mockServer.mu.RUnlock()

	fmt.Printf("\n📊 Statistics:\n")
	fmt.Printf("  - Total logs received by mock server: %d\n", totalLogs)
	fmt.Printf("  - Total requests: %d\n", mockServer.requestCount)
	fmt.Printf("  - Success count: %d\n", mockServer.successCount)
	fmt.Printf("  - Error count: %d\n", mockServer.errorCount)
	fmt.Printf("  - Total bytes: %d\n", mockServer.totalBytes)
	fmt.Printf("  - Uptime: %v\n", time.Since(mockServer.startTime))

	// 7. 显示收到的日志
	if totalLogs > 0 {
		fmt.Println("\n📋 Received logs:")
		mockServer.mu.RLock()
		for i, entry := range mockServer.logs {
			if msg, ok := entry.Data["message"].(string); ok {
				fmt.Printf("  [%d] %s | service=%v level=%v\n",
					i, msg, entry.Data["service"], entry.Data["level"])
			}
		}
		mockServer.mu.RUnlock()
	}

	fmt.Println("\n✅ Integration test completed successfully!")
	fmt.Printf("\n💡 You can also check:\n")
	fmt.Printf("  - Mock server stats: http://localhost:%d/stats\n", config.Port)
	fmt.Printf("  - View all logs: http://localhost:%d/logs\n", config.Port)
}
