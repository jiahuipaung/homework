# Task JSON Builder

通过 Go 代码可编程地生成 fc-stash 测试用的 task JSON 配置文件，无需依赖前端配置页面。

## 功能特性

- ✅ 类型安全的 Go 结构体
- ✅ 流畅的 Builder API
- ✅ 预设的 VMLogs 任务模板
- ✅ 支持 Kafka 和 VictoriaLogs 数据源
- ✅ 支持字段转换和重命名
- ✅ 一键生成 JSON 配置文件

## 快速开始

### 最简单的方式

```go
package main

import (
    "github.com/flashcatcloud/fc-stash/mockserver/taskbuilder"
)

func main() {
    // 一行代码创建完整配置
    config := taskbuilder.QuickVMLogsConfig(
        []string{"10.99.1.105:9092"},       // Kafka brokers
        "fc-insight-self",                   // Kafka topic
        "vmlogs-test-group",                 // Kafka consumer group
        []string{"http://10.99.1.15:9428"}, // VMLogs endpoints
        "root",                              // VMLogs user
        "root.2020",                         // VMLogs password
    )

    // 保存到文件
    config.SaveToFile("test.json")
}
```

### 分步构建

```go
config := taskbuilder.NewConfigBuilder().
    AddKafkaSource("20001",
        []string{"10.99.1.105:9092"},
        "fc-insight-self",
        "vmlogs-test-group").
    AddVMLogsSource("20002",
        []string{"http://10.99.1.15:9428"},
        "root",
        "root.2020").
    AddVMLogsTaskSimple(1001, 20001, 20002).
    Build()

config.SaveToFile("test.json")
```

### 完全自定义

```go
// 自定义 Task 构建
task := taskbuilder.NewTaskBuilder().
    WithID(1001).
    WithDataSourceID(20001).
    WithPrefixMatch("message", "level", "timestamp", "service_name").
    AddTimestampFormat("timestamp", "%s%3N", "Local").
    AddFieldRename("timestamp", "@timestamp", "float", "text").
    WithVMLogsStorage(20002, "message", "@timestamp",
        []string{"service_name", "level", "hostname"}).
    Build()

// 构建完整配置
config := taskbuilder.NewConfigBuilder().
    AddKafkaSource("20001", []string{"10.99.1.105:9092"}, "topic", "group").
    AddVMLogsSource("20002", []string{"http://10.99.1.15:9428"}, "root", "root.2020").
    AddTask(task).
    Build()

config.SaveToFile("custom-test.json")
```

## API 参考

### ConfigBuilder

主配置构建器，用于管理数据源和任务。

| 方法 | 说明 |
|------|------|
| `NewConfigBuilder()` | 创建新的配置构建器 |
| `AddKafkaSource(id, brokers, topic, group)` | 添加 Kafka 数据源 |
| `AddVMLogsSource(id, endpoints, user, password)` | 添加 VMLogs 数据源 |
| `AddTask(task)` | 添加任务 |
| `AddVMLogsTaskSimple(taskID, kafkaID, vmlogsID)` | 添加简单 VMLogs 任务 |
| `AddVMLogsTaskCustom(config)` | 添加自定义 VMLogs 任务 |
| `Build()` | 构建最终配置 |
| `ToJSON()` | 转换为 JSON 字符串 |
| `SaveToFile(filename)` | 保存到文件 |

### TaskBuilder

任务构建器，用于创建单个任务。

| 方法 | 说明 |
|------|------|
| `NewTaskBuilder()` | 创建新的任务构建器 |
| `WithID(id)` | 设置任务 ID |
| `WithDataSourceID(id)` | 设置输入数据源 ID |
| `WithPrefixMatch(fields...)` | 设置提取字段列表 |
| `AddTimestampFormat(field, format, location)` | 添加时间戳格式化 |
| `AddFieldRename(from, to, fromType, toType)` | 添加字段重命名 |
| `AddFieldTransform(transform)` | 添加自定义字段转换 |
| `WithVMLogsStorage(id, msgField, timeField, streamFields)` | 配置 VMLogs 存储 |
| `Build()` | 构建最终任务 |

### 辅助函数

| 函数 | 说明 |
|------|------|
| `QuickVMLogsConfig(...)` | 快速创建 VMLogs 配置 |
| `NewDefaultVMLogsTask(taskID, kafkaID, vmlogsID)` | 创建默认 VMLogs 任务 |
| `NewVMLogsTask(config)` | 从配置创建 VMLogs 任务 |

## 运行示例

```bash
# 进入示例目录
cd mockserver/taskbuilder/example

# 运行示例程序
go run main.go

# 查看生成的文件
ls -lh *.json
```

示例程序会生成 3 个配置文件：
- `quick-test.json` - 最简单的配置
- `step-by-step-test.json` - 分步构建的配置
- `custom-test.json` - 完全自定义的配置

## 使用场景

### 1. 集成测试

在测试代码中动态生成配置：

```go
func TestVMLogsIntegration(t *testing.T) {
    // 生成测试配置
    config := taskbuilder.QuickVMLogsConfig(
        []string{"localhost:9092"},
        "test-topic",
        "test-group",
        []string{"http://localhost:9428"},
        "test",
        "test",
    )
    config.SaveToFile("test-config.json")

    // 启动 mockserver 使用这个配置
    // ... 测试逻辑
}
```

### 2. 不同场景的配置

为不同的测试场景创建不同的配置：

```go
// 场景 1: 单 broker, 单 endpoint
scenario1 := taskbuilder.QuickVMLogsConfig(...)
scenario1.SaveToFile("scenario1.json")

// 场景 2: 多 broker, 多 endpoint
config2 := taskbuilder.NewConfigBuilder().
    AddKafkaSource("20001",
        []string{"broker1:9092", "broker2:9092"},
        "topic", "group").
    AddVMLogsSource("20002",
        []string{"http://vmlogs1:9428", "http://vmlogs2:9428"},
        "user", "pass").
    AddVMLogsTaskSimple(1001, 20001, 20002)
config2.SaveToFile("scenario2.json")
```

### 3. 自定义字段映射

测试不同的字段提取和转换规则：

```go
task := taskbuilder.NewTaskBuilder().
    WithID(1001).
    WithDataSourceID(20001).
    // 提取自定义字段
    WithPrefixMatch("msg", "lvl", "ts", "svc", "host").
    // 自定义时间戳处理
    AddTimestampFormat("ts", "%Y-%m-%d %H:%M:%S", "UTC").
    AddFieldRename("ts", "@timestamp", "string", "text").
    AddFieldRename("msg", "message", "string", "text").
    AddFieldRename("lvl", "level", "string", "text").
    // 自定义 stream fields
    WithVMLogsStorage(20002, "message", "@timestamp",
        []string{"svc", "lvl", "host"}).
    Build()
```

## 与 Mock Config Server 集成

生成的 JSON 文件可以直接用于 Mock Config Server：

```bash
# 1. 生成配置文件
cd mockserver/taskbuilder/example
go run main.go

# 2. 复制到 config-server 目录（或使用绝对路径）
cp quick-test.json ../../config-server/test.json

# 3. 启动 Mock Config Server
cd ../../config-server
go run main.go -config test.json

# 4. fc-stash 从 Mock Server 获取配置
# 配置 fc-stash 的 InsightService 地址指向 http://localhost:8080
```

## 生成的 JSON 结构

生成的 JSON 符合 fc-stash 期望的格式：

```json
{
  "data": {
    "data_sources": {
      "20001": {
        "type": "kafka",
        "settings": { ... }
      },
      "20002": {
        "type": "victoria_logs",
        "settings": { ... }
      }
    },
    "tasks": [
      {
        "id": 1001,
        "data_source_id": 20001,
        "settings": { ... },
        "storage": {
          "data_source_type": "victoria_logs",
          "victoria_logs": { ... }
        }
      }
    ],
    "label_mappings": {}
  }
}
```

## 注意事项

1. **数据源 ID**: 建议使用 5 位数字（如 20001, 20002）以避免与真实环境冲突
2. **任务 ID**: 建议使用 4 位数字（如 1001, 1002）
3. **Stream Fields**: VMLogs 的 stream fields 应该选择基数不太高的字段
4. **时间字段**: 确保 timestamp 字段格式与 VMLogs 期望一致

## 下一步

- [ ] 添加 Elasticsearch 存储支持
- [ ] 添加 Doris 存储支持
- [ ] 添加更多字段转换规则（正则、脱敏等）
- [ ] 添加配置验证功能
