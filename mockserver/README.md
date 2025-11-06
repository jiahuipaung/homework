# VMLogs MockServer - 集成测试工具

## 📋 概述

这是一个用于 `sink/vmlogs` 模块集成测试的 Mock Server 工具，**模拟 fc-insight/n9e-plus 配置服务端**，让你可以使用真实的配置 JSON 对整个 fc-stash 系统进行集成测试。

## 🎯 测试架构

```
┌──────────────────┐
│  Config Server   │  提供配置 API（你要实现的）
│  (mockserver)    │  - GET /api/v2/dimensions/logevent/tasks
│                  │  - GET /v1/n9e-plus/logevent/tasks
└────────┬─────────┘
         │ 返回配置 JSON
         ▼
┌──────────────────┐
│    fc-stash      │  正常运行（完全不用改代码！）
│    (真实程序)     │  - 从 Config Server 获取配置
│                  │  - 从生产 Kafka 消费数据
│                  │  - 写入测试环境 VictoriaLogs
└────────┬─────────┘
         │
         ▼
   生产 Kafka → Extract → 测试 VictoriaLogs
   (真实)      (真实)      (真实)
```

## 🚀 快速开始

### 1. 启动 Config Server

```bash
cd mockserver/config-server

# 使用 VMLogs 测试配置
go run main.go

# 使用你的生产配置
go run main.go -config ../configs/production-tasks.json -verbose
```

**启动成功后看到：**
```
🚀 Mock Config Server started on http://localhost:8080
📝 Config file: configs/vmlogs-test-tasks.json
📋 Endpoints:
   - InsightService: http://localhost:8080/api/v2/dimensions/logevent/tasks
   - N9eService:     http://localhost:8080/v1/n9e-plus/logevent/tasks
   - Status:         http://localhost:8080/status
🔄 Hot reload: Enabled
```

### 2. 配置 fc-stash 指向 Mock Server

修改 `etc/stash.yml`：

```yaml
stash:
  insight_service:
    webapis:
    - http://localhost:8080  # 指向 Mock Server
```

### 3. 启动 fc-stash

```bash
./stash -c etc/ -l logs/

# fc-stash 会自动从 Mock Server 获取配置并运行！
```

## 📁 配置文件

### configs/vmlogs-test-tasks.json

VMLogs 测试配置示例：

```json
{
  "data": {
    "data_sources": {
      "20001": {
        "type": "kafka",
        "settings": {
          "kafka.brokers": ["10.99.1.105:9092"],
          "kafka.topic": "your-topic",
          "kafka.group": "test-group"
        }
      },
      "20002": {
        "type": "victoria_logs",
        "settings": {
          "vmlogs.endpoints": ["http://10.99.1.15:9428"],
          "vmlogs.user": "root",
          "vmlogs.password": "root.2020"
        }
      }
    },
    "tasks": [...]
  }
}
```

### configs/production-tasks.json

你的生产环境真实配置（简化版）。**你可以把完整的生产配置放这里！**

## 🔧 修改配置进行测试

### 支持热重载！

直接修改配置文件，2秒内自动生效：

```bash
vim configs/vmlogs-test-tasks.json
# 修改 stream_fields、msg_field 等配置
# 保存后自动重载！
```

或手动触发：
```bash
curl -X POST http://localhost:8080/reload
```

### 测试不同场景

#### 场景1: 测试自动字段
```json
{
  "msg_field_settings": { "auto_msg": true },
  "time_field_settings": { "auto_time": true }
}
```

#### 场景2: 测试自定义 stream_fields
```json
{
  "stream_fields_settings": {
    "stream_field_names": ["service", "level", "host"]
  }
}
```

#### 场景3: 测试不同时间字段
```json
{
  "time_field_settings": {
    "auto_time": false,
    "time_field_name": "log_timestamp"
  }
}
```

## 📊 监控和验证

### 查看 Mock Server 状态

```bash
curl http://localhost:8080/status
```

### 测试 API

```bash
# 测试 InsightService API
curl http://localhost:8080/api/v2/dimensions/logevent/tasks | jq

# 测试 N9eService API
curl -u user:pass http://localhost:8080/v1/n9e-plus/logevent/tasks | jq
```

### 查看 fc-stash 日志

```bash
tail -f logs/stash.log | grep vmlogs
```

### 验证 VictoriaLogs 数据

```bash
curl -G 'http://10.99.1.15:9428/select/logsql/query' \
  --data-urlencode 'query={service="test"}' \
  -u root:root.2020
```

## 🧪 完整测试流程

```bash
# 1. 启动 Mock Server
cd mockserver/config-server
go run main.go -verbose

# 2. 修改 fc-stash 配置
vim ../../etc/stash.yml  # 设置 insight_service.webapis

# 3. 启动 fc-stash
cd ../..
./stash -c etc/ -l logs/

# 4. 观察日志
tail -f logs/stash.log

# 5. 修改配置测试
vim mockserver/configs/vmlogs-test-tasks.json
# 保存后自动生效！等待 fc-stash 拉取（约2分钟）

# 6. 验证数据
curl -G 'http://10.99.1.15:9428/select/logsql/query' \
  --data-urlencode 'query={fcservice="insight"}' \
  -u root:root.2020
```

## 🔍 故障排查

### Config Server 无法启动
```bash
# 检查配置文件
cat configs/vmlogs-test-tasks.json | jq .

# 检查端口
lsof -i :8080
```

### fc-stash 无法获取配置
```bash
# 测试 Mock Server
curl http://localhost:8080/status

# 检查 fc-stash 配置
cat etc/stash.yml | grep -A 3 insight_service

# 查看日志
tail -f logs/stash.log | grep "insight service"
```

### 数据未写入 VictoriaLogs
```bash
# 检查 VictoriaLogs
curl http://10.99.1.15:9428/health

# 查看 vmlogs 日志
tail -f logs/stash.log | grep vmlogs

# 检查配置
cat configs/vmlogs-test-tasks.json | jq '.data.data_sources."20002"'
```

## 💡 使用技巧

1. **配置版本控制**：把测试配置加入 Git
2. **多场景配置**：准备多个配置文件测试不同场景
3. **增量测试**：从简单配置开始，逐步增加复杂度
4. **日志对比**：对比修改前后的日志和数据

## 📝 命令行参数

```bash
go run main.go [options]

Options:
  -config string
        配置文件路径 (default "configs/vmlogs-test-tasks.json")
  -port int
        HTTP 端口 (default 8080)
  -verbose
        详细日志 (default false)
  -hot-reload
        热重载配置 (default true)
```

## 🎓 进阶：混合测试

同时测试 ES、Doris、VMLogs：

```json
{
  "tasks": [
    { "storage": { "elasticsearch": {...} } },
    { "storage": { "doris": {...} } },
    { "storage": { "victoria_logs": {...} } }
  ]
}
```

fc-stash 会同时向三种目标写入！

## 📚 相关链接

- [VictoriaLogs 文档](https://docs.victoriametrics.com/victorialogs/)
- [sink/vmlogs 源码](../sink/vmlogs/)

---

**这就是你需要的！** 通过 Mock Config Server，你可以：
- ✅ 脱离前端，直接测试后端
- ✅ 使用真实配置 JSON
- ✅ 快速修改配置进行各种测试
- ✅ 完全不用改 fc-stash 代码
