package processor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"goserver/internal/cache"
	"goserver/internal/config"
)

// DataProcessor 数据处理器
type DataProcessor struct {
	redis      *cache.RedisClient
	config     *config.Config
	workerPool chan struct{}
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	metrics    *ProcessorMetrics
}

// ProcessorMetrics 处理器指标
type ProcessorMetrics struct {
	mu              sync.RWMutex
	ProcessedCount  int64
	FailedCount     int64
	AverageLatency  float64
	TotalLatency    int64
	LastProcessTime time.Time
}

// NewDataProcessor 创建新的数据处理器
func NewDataProcessor(redis *cache.RedisClient, cfg *config.Config) *DataProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	return &DataProcessor{
		redis:      redis,
		config:     cfg,
		workerPool: make(chan struct{}, cfg.App.Workers),
		ctx:        ctx,
		cancel:     cancel,
		metrics:    &ProcessorMetrics{},
	}
}

// Process 处理单个数据项
func (dp *DataProcessor) Process(data interface{}) error {
	dp.wg.Add(1)
	defer dp.wg.Done()

	// 获取一个worker令牌
	dp.workerPool <- struct{}{}
	defer func() { <-dp.workerPool }()

	start := time.Now()

	// 模拟处理业务逻辑
	err := dp.processData(data)
	
	latency := time.Since(start).Milliseconds()
	
	dp.metrics.mu.Lock()
	defer dp.metrics.mu.Unlock()
	
	if err != nil {
		dp.metrics.FailedCount++
		return err
	}
	
	dp.metrics.ProcessedCount++
	dp.metrics.TotalLatency += latency
	dp.metrics.LastProcessTime = time.Now()
	
	if dp.metrics.ProcessedCount > 0 {
		dp.metrics.AverageLatency = float64(dp.metrics.TotalLatency) / float64(dp.metrics.ProcessedCount)
	}

	return nil
}

// processData 实际的数据处理逻辑
func (dp *DataProcessor) processData(data interface{}) error {
	ctx, cancel := context.WithTimeout(dp.ctx, 5*time.Second)
	defer cancel()

	// 处理逻辑示例：将数据保存到Redis
	key := fmt.Sprintf("data:%d", time.Now().UnixNano())
	return dp.redis.Set(ctx, key, data, 24*time.Hour)
}

// ProcessBatch 批量处理数据
func (dp *DataProcessor) ProcessBatch(dataList []interface{}) (int, error) {
	successCount := 0
	
	for _, data := range dataList {
		if err := dp.Process(data); err != nil {
			continue
		}
		successCount++
	}

	return successCount, nil
}

// ProcessFromQueue 从Redis队列处理数据
func (dp *DataProcessor) ProcessFromQueue(queueKey string, batchSize int) (int, error) {
	ctx := context.Background()
	
	processed := 0
	for i := 0; i < batchSize; i++ {
		val, err := dp.redis.Rpop(ctx, queueKey)
		if err != nil {
			if err.Error() == "redis: nil" {
				break
			}
			return processed, err
		}

		_ = dp.Process(val)
		processed++
	}

	return processed, nil
}

// GetMetrics 获取处理指标
func (dp *DataProcessor) GetMetrics() ProcessorMetrics {
	dp.metrics.mu.RLock()
	defer dp.metrics.mu.RUnlock()
	return *dp.metrics
}

// ResetMetrics 重置指标
func (dp *DataProcessor) ResetMetrics() {
	dp.metrics.mu.Lock()
	defer dp.metrics.mu.Unlock()
	dp.metrics.ProcessedCount = 0
	dp.metrics.FailedCount = 0
	dp.metrics.TotalLatency = 0
	dp.metrics.AverageLatency = 0
}

// Stop 停止处理器
func (dp *DataProcessor) Stop() {
	dp.cancel()
	dp.wg.Wait()
}

// Wait 等待所有处理完成
func (dp *DataProcessor) Wait() {
	dp.wg.Wait()
}
