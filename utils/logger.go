package utils

import "go.uber.org/zap"

var (
	Logger *zap.Logger
)

func InitLog() error {
	logger, err := zap.NewProduction()
	if err != nil {
		return err
	}
	Logger = logger
	return nil
}
