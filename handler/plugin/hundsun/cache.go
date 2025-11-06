package hundsun

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/flashcatcloud/fc-stash/utils"
	"github.com/flashcatcloud/go-pkg/srv"
)

// 定长随机丢弃
// 用于本地缓存request_log, 目前没有想到更好的丢弃策略
type FixedSizeRandomDropCache struct {
	sync.RWMutex
	Size  int
	Cache map[string]*RequestExtractedLog
}

func NewFixedSizeRandomDropCache(size int) *FixedSizeRandomDropCache {
	return &FixedSizeRandomDropCache{
		Size:  size,
		Cache: make(map[string]*RequestExtractedLog),
	}
}

// 超过缓存大小就随机丢掉 1/10
func (c *FixedSizeRandomDropCache) Set(key string, value *RequestExtractedLog) {
	c.Lock()
	defer c.Unlock()
	if len(c.Cache) >= c.Size {
		max := len(c.Cache) / 10
		for key := range c.Cache { // map range是随机的
			delete(c.Cache, key)
			max--
			if max <= 0 {
				break
			}
		}
	}
	c.Cache[key] = value
}

func (c *FixedSizeRandomDropCache) GetAndRemove(key string, remove bool) (*RequestExtractedLog, bool) {
	c.Lock()
	defer c.Unlock()

	req, found := c.Cache[key]
	if found {
		if remove {
			delete(c.Cache, key)
		}
		return req, true
	}
	return nil, false
}

// worker_id + no + agent_hostname
func EncodeCachedKey(ctx *srv.Context, log *RequestExtractedLog, unique string) string {
	return "logx_hundsun_" +
		utils.EncodeUUIDWithSortedKeys([]string{unique, log.ReqNo, log.AgentHostname})
}

// ttl: DuplicateWindowSeconds
func (p *PipelineWorker) WriteToCacheBackend(ctx *srv.Context, log *RequestExtractedLog) error {
	if log == nil {
		return errors.New("nil pointer: request_log")
	}
	req := &RequestCachedLog{
		UnixMilli:   log.Date.UnixMilli(),
		FundAccount: log.FundAccount,
		Tag:         log.Tag,
		Data:        log.Data,
	}
	data, _ := json.Marshal(req)
	return p.remoteCache.Set(ctx,
		EncodeCachedKey(ctx, log, p.WorkerID),
		data,
		time.Second*time.Duration(p.DuplicateWindowSeconds),
	).Err()
}
