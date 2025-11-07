package main

import (
	"fmt"
	"log"

	"github.com/flashcatcloud/fc-stash/mockserver/taskbuilder"
)

func main() {
	fmt.Println("🚀 VMLogs Task JSON Generator")
	fmt.Println("==============================\n")

	// Example 1: Quick setup - simplest way
	example1()

	// Example 2: Step-by-step setup with more control
	example2()

	// Example 3: Fully customized task
	example3()
}

// example1: Quickest way to create a basic VMLogs test config
func example1() {
	fmt.Println("📝 Example 1: Quick VMLogs Config")
	fmt.Println("-----------------------------------")

	config := taskbuilder.QuickVMLogsConfig(
		[]string{"10.99.1.105:9092"},           // Kafka brokers
		"fc-insight-self",                       // Kafka topic
		"vmlogs-test-group",                     // Kafka consumer group
		[]string{"http://10.99.1.15:9428"},     // VMLogs endpoints
		"root",                                  // VMLogs user
		"root.2020",                             // VMLogs password
	)

	if err := config.SaveToFile("quick-test.json"); err != nil {
		log.Fatalf("Failed to save config: %v", err)
	}

	fmt.Println("✅ Generated: quick-test.json")
	fmt.Println("   - 1 Kafka source (ID: 20001)")
	fmt.Println("   - 1 VMLogs source (ID: 20002)")
	fmt.Println("   - 1 VMLogs task (ID: 1001)")
	fmt.Println()
}

// example2: Step-by-step configuration with builder
func example2() {
	fmt.Println("📝 Example 2: Step-by-Step Configuration")
	fmt.Println("------------------------------------------")

	config := taskbuilder.NewConfigBuilder().
		// Add Kafka input source
		AddKafkaSource("20001",
			[]string{"10.99.1.105:9092"},
			"fc-insight-self",
			"vmlogs-test-group",
		).
		// Add VMLogs output source
		AddVMLogsSource("20002",
			[]string{"http://10.99.1.15:9428"},
			"root",
			"root.2020",
		).
		// Add a simple VMLogs task
		AddVMLogsTaskSimple(1001, 20001, 20002)

	if err := config.SaveToFile("step-by-step-test.json"); err != nil {
		log.Fatalf("Failed to save config: %v", err)
	}

	fmt.Println("✅ Generated: step-by-step-test.json")
	fmt.Println("   - Same structure as Example 1")
	fmt.Println("   - But with explicit step-by-step building")
	fmt.Println()
}

// example3: Fully customized task with custom fields and transformations
func example3() {
	fmt.Println("📝 Example 3: Fully Customized Task")
	fmt.Println("------------------------------------")

	// Build custom task with specific configuration
	customTask := taskbuilder.NewTaskBuilder().
		WithID(2001).
		WithDataSourceID(20001).
		// Extract these fields from JSON
		WithPrefixMatch("message", "level", "timestamp", "service_name", "hostname").
		// Format timestamp field
		AddTimestampFormat("timestamp", "%s%3N", "Local").
		// Rename timestamp to @timestamp
		AddFieldRename("timestamp", "@timestamp", "float", "text").
		// Configure VMLogs storage with custom stream fields
		WithVMLogsStorage(
			20002,                                  // VMLogs source ID
			"message",                              // Message field
			"@timestamp",                           // Time field
			[]string{"service_name", "level", "hostname"}, // Stream fields
		).
		Build()

	// Build complete config with multiple sources and tasks
	config := taskbuilder.NewConfigBuilder().
		// Kafka source with custom settings
		AddKafkaSource("20001",
			[]string{"10.99.1.105:9092", "10.99.1.106:9092"}, // Multiple brokers
			"custom-topic",
			"custom-group",
		).
		// VMLogs source
		AddVMLogsSource("20002",
			[]string{"http://10.99.1.15:9428"},
			"root",
			"root.2020",
		).
		// Add the custom task
		AddTask(customTask).
		// Can add more tasks
		AddVMLogsTaskSimple(2002, 20001, 20002)

	if err := config.SaveToFile("custom-test.json"); err != nil {
		log.Fatalf("Failed to save config: %v", err)
	}

	fmt.Println("✅ Generated: custom-test.json")
	fmt.Println("   - 1 Kafka source with 2 brokers")
	fmt.Println("   - 1 VMLogs source")
	fmt.Println("   - 2 tasks: one custom (ID: 2001), one default (ID: 2002)")
	fmt.Println("   - Custom stream fields: service_name, level, hostname")
	fmt.Println()
}
