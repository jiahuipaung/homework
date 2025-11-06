package g_account

import (
	"errors"
	"strconv"
	"time"

	"github.com/flashcatcloud/go-pkg/srv"
	"github.com/flashcatcloud/go-pkg/x/redisx"
)

// 由于是动态词表, 需要维护在中心端, 这里选择的是redis
// 词表是硬编码的, AccountRef -> AccountID
// 词表每隔一段时间会重置

const (
	KeyEnrichDictGAccountPrefix = "logx_enrich_g_account_"
)

// logx_enrich_g_account_{ref}_latest 中存储的是 下一步查询对应的key, 下一步查出来的才是真实的value
// logx_enrich_g_account_{ref}_{unixsec} 中存储的是 真实的value
// 两个key的操作暂时不加锁
func SetDictGAccountRefToID(ctx *srv.Context, hostname string, ref string, ID string,
	t time.Time, ttl time.Duration) error {
	rediscli, err := redisx.Get(ctx)
	if err != nil {
		return err
	}
	latest, curr := redisKeyEnrichDictGAccountRef(hostname, ref, t.Unix())
	if err := rediscli.Set(ctx, curr, ID, ttl).Err(); err != nil {
		return err
	}
	if err := rediscli.Set(ctx, latest, curr, ttl).Err(); err != nil {
		return err
	}
	return nil
}

// 上层没有拿到映射的时候, 需要"等待一会儿"再重新查一次, 避免因延迟导致的问题
func GetDictGAccountIDByRef(ctx *srv.Context, hostname, ref string) (ID string, err error) {
	rediscli, err := redisx.Get(ctx)
	if err != nil {
		return
	}
	latest, _ := redisKeyEnrichDictGAccountRef(hostname, ref, 0)
	curr, err := rediscli.Get(ctx, latest).Result()
	if err != nil {
		// 只有 NilErr 时上层需要重试
		// 其他错误上层直接退出
		return
	}
	if len(curr) == 0 {
		err = errors.New("fatal error")
		return
	}
	ID, err = rediscli.Get(ctx, curr).Result()
	if err != nil {
		// NilErr 时上层也需要重试
		// 其他错误上层直接退出
		return
	}
	return
}

func redisKeyEnrichDictGAccountRef(hostname, ref string, unixsec int64) (latest string, curr string) {
	prefix := KeyEnrichDictGAccountPrefix + hostname + "_" + ref
	return prefix + "_latest", prefix + "_" + strconv.FormatInt(unixsec, 10)
}
