package doris

import (
	"sync"
	"time"

	"github.com/flashcatcloud/fc-stash/types"
)

type BatchProcessor struct {
	buffer    []types.ExtractedLog
	bufSize   int
	interval  time.Duration
	processor func([]types.ExtractedLog) error

	mu     sync.Mutex
	ticker *time.Ticker
	done   chan struct{}
}

func NewBatchProcessor(size int, interval time.Duration,
	processor func([]types.ExtractedLog) error) *BatchProcessor {

	bp := &BatchProcessor{
		buffer:    make([]types.ExtractedLog, 0, size),
		bufSize:   size,
		interval:  interval,
		processor: processor,
		done:      make(chan struct{}),
	}

	bp.ticker = time.NewTicker(interval)
	go bp.run()
	return bp
}

// Add 在达到阈值时刷缓存数据
func (bp *BatchProcessor) Add(item types.ExtractedLog) error {
	bp.mu.Lock()
	bp.buffer = append(bp.buffer, item)

	if len(bp.buffer) >= bp.bufSize {
		batch := bp.buffer
		bp.buffer = make([]types.ExtractedLog, 0, bp.bufSize)
		bp.mu.Unlock()
		return bp.processor(batch)
	}

	bp.mu.Unlock()
	return nil
}

// run 周期性地刷缓存数据
func (bp *BatchProcessor) run() {
	for {
		select {
		case <-bp.ticker.C:
			bp.mu.Lock()
			if len(bp.buffer) > 0 {
				batch := bp.buffer
				bp.buffer = make([]types.ExtractedLog, 0, bp.bufSize)
				bp.mu.Unlock()
				bp.processor(batch)
			} else {
				bp.mu.Unlock()
			}
		case <-bp.done:
			return
		}
	}
}

// Close 在关闭bp时刷缓存数据
func (bp *BatchProcessor) Close() {
	bp.ticker.Stop()
	close(bp.done)

	bp.mu.Lock()
	if len(bp.buffer) > 0 {
		bp.processor(bp.buffer)
	}
	bp.mu.Unlock()
}
