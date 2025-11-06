# VMLogs MockServer - WIP

## 状态

⚠️ **工作进行中 (Work In Progress)**

这是一个用于测试 `sink/vmlogs` 模块的 mockserver 初始版本。

## 当前实现

### 已完成
- ✅ VictoriaLogs API 模拟 (`POST /insert/jsonline`)
- ✅ 健康检查接口 (`GET /health`)
- ✅ 管理接口（stats, logs, control）
- ✅ 基础集成测试示例

### 待讨论和确认

需要根据实际需求调整测试方案：

1. **测试范围**
   - 是否需要测试完整 Pipeline（Kafka → Extract → VMLogs）
   - 还是只测试 VMLogs 输出模块

2. **配置数据来源**
   - 使用手动准备的 JSON 配置文件
   - 还是调用真实的 InsightService/N9eService API

3. **Mock 组件**
   - 是否需要 Mock Kafka 输入
   - 是否需要 Mock InsightService/N9eService API

4. **测试目标**
   - 功能验证
   - 性能测试
   - 回归测试

## 文件说明

- `main.go` - MockServer 主程序入口
- `server.go` - HTTP 服务器实现（模拟 VictoriaLogs API）
- `integration_test.go` - 集成测试用例（直接测试 VMLogsOutput）
- `example.go` - 使用示例

## 下一步

等待需求确认后，根据实际情况调整：
- 可能需要添加配置文件加载
- 可能需要添加 Kafka Mock
- 可能需要添加完整 Pipeline 测试

## 使用方法

暂时不建议使用，等待需求确认和代码调整。
