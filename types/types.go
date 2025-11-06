package types

import (
	"context"

	"go.uber.org/zap"
)

type ExtractedLog map[string]interface{}

type LogEvent struct {
	Source  string      `json:"source"`  // 消息来源
	Message interface{} `json:"message"` // 消息原文, 只支持 string和map[string]interface{} 结构
	Errors  []string    `json:"errors"`  // 是否出错
}

// 输入, 数据写入到缓存队列中
type Input interface {
	Init(ctx context.Context) error
	Start(ctx context.Context, logger *zap.Logger, queue chan<- *LogEvent) error
	Stop(ctx context.Context)
}

// Deprecated, 旧版本依赖
type FieldParser interface {
	ParseString(origin string) (string, error)
}

type TextParser interface {
	Name() string
	Parse(origin string) (map[string]interface{}, error)
}

// 提取
type Extract interface {
	Init(ctx context.Context) error
	Handle(ctx context.Context, e *LogEvent) (ExtractedLog, error)
	GetUUID() string
}

type Output interface {
	Name() string
	Init(ctx context.Context, worker int) error
	Start(ctx context.Context, logger *zap.Logger) error
	Stop(ctx context.Context)
	Sink(ctx context.Context, ext ExtractedLog) error
	IsDiff(other Output) bool
}
