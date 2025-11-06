package pipeline_handler

import (
	"github.com/flashcatcloud/fc-stash/config"
	"github.com/flashcatcloud/fc-stash/utils"
	"go.uber.org/zap"
)

var (
	conf config.HandlerConfig
)

// init conf and logger
func SetConfig(newest config.HandlerConfig, newlogger *zap.Logger) {
	conf = newest
	if newlogger != nil {
		logger = newlogger
		utils.Logger = newlogger
	}
}
