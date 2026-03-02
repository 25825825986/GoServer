// Package processor 业务处理器 - 支持消息队列发布订阅
package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"goserver/internal/cache"
	"goserver/internal/config"
	"goserver/internal/protocol"
)

// DataProcessor 数据处理器
type DataProcessor struct {
	redis      *cache.RedisClient
	config     *config.Config
	
	// 订阅管理
	subscribers   map[string]map[string]protocol.ConnInterface // channel -> connID -> connection
	subscribersMu sync.RWMutex
	
	// 指标
	metrics *ProcessorMetrics
}

// ProcessorMetrics 处理器指标
type ProcessorMetrics struct {
	mu              sync.RWMutex
	ProcessedCount  uint64
	FailedCount     uint64
	PublishedCount  uint64
	SubscribedCount uint64
	TotalLatency    int64
	AvgLatency      float64
	LastProcessTime time.Time
}

// NewDataProcessor 创建新的数据处理器
func NewDataProcessor(redis *cache.RedisClient, cfg *config.Config) *DataProcessor {
	return &DataProcessor{
		redis:       redis,
		config:      cfg,
		subscribers: make(map[string]map[string]protocol.ConnInterface),
		metrics:     &ProcessorMetrics{},
	}
}

// Handle 实现网络处理器接口 - 处理各种命令
func (dp *DataProcessor) Handle(connID string, conn protocol.ConnInterface, msg *protocol.Message) *protocol.Response {
	start := time.Now()
	
	var resp *protocol.Response
	
	switch msg.Cmd {
	case protocol.CmdProcess:
		resp = dp.handleProcess(msg)
	case protocol.CmdBatch:
		resp = dp.handleBatch(msg)
	case protocol.CmdPublish:
		resp = dp.handlePublish(msg)
	case protocol.CmdSubscribe:
		resp = dp.handleSubscribe(connID, conn, msg)
	case protocol.CmdGetMetrics:
		resp = dp.handleGetMetrics(msg)
	case protocol.CmdGetConfig:
		resp = dp.handleGetConfig(msg)
	default:
		resp = protocol.NewErrorResponse(msg.ID, fmt.Sprintf("unknown command: %s", msg.Cmd))
	}

	// 更新指标
	latency := time.Since(start).Milliseconds()
	resp.Latency = latency
	dp.updateMetrics(resp.Status == "ok", latency)

	return resp
}

// handleProcess 处理单条数据
func (dp *DataProcessor) handleProcess(msg *protocol.Message) *protocol.Response {
	if msg.Data == nil {
		return protocol.NewErrorResponse(msg.ID, "data is nil")
	}

	// 存储到Redis
	key := fmt.Sprintf("data:%d", time.Now().UnixNano())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	jsonData, err := json.Marshal(msg.Data)
	if err != nil {
		return protocol.NewErrorResponse(msg.ID, fmt.Sprintf("marshal error: %v", err))
	}

	if err := dp.redis.Set(ctx, key, string(jsonData), 24*time.Hour); err != nil {
		return protocol.NewErrorResponse(msg.ID, fmt.Sprintf("redis error: %v", err))
	}

	return protocol.NewResponse(msg.ID, map[string]string{
		"status": "processed",
		"key":    key,
	})
}

// handleBatch 批量处理
func (dp *DataProcessor) handleBatch(msg *protocol.Message) *protocol.Response {
	dataList, ok := msg.Data.([]interface{})
	if !ok {
		return protocol.NewErrorResponse(msg.ID, "data must be an array")
	}

	successCount := 0
	failedCount := 0
	keys := make([]string, 0, len(dataList))

	for _, item := range dataList {
		key := fmt.Sprintf("data:%d", time.Now().UnixNano())
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		
		jsonData, err := json.Marshal(item)
		if err != nil {
			failedCount++
			cancel()
			continue
		}

		if err := dp.redis.Set(ctx, key, string(jsonData), 24*time.Hour); err != nil {
			failedCount++
			cancel()
			continue
		}
		cancel()

		successCount++
		keys = append(keys, key)
	}

	return protocol.NewResponse(msg.ID, map[string]interface{}{
		"total":         len(dataList),
		"success_count": successCount,
		"failed_count":  failedCount,
		"keys":          keys,
	})
}

// handlePublish 发布消息到队列（支持Redis List和普通订阅）
func (dp *DataProcessor) handlePublish(msg *protocol.Message) *protocol.Response {
	if msg.Channel == "" {
		return protocol.NewErrorResponse(msg.ID, "channel is required")
	}

	// 1. 存储到Redis List（作为消息队列）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	jsonData, err := json.Marshal(msg.Data)
	if err != nil {
		return protocol.NewErrorResponse(msg.ID, fmt.Sprintf("marshal error: %v", err))
	}

	// 使用 LPUSH 将消息推入队列
	if _, err := dp.redis.Lpush(ctx, msg.Channel, string(jsonData)); err != nil {
		return protocol.NewErrorResponse(msg.ID, fmt.Sprintf("redis error: %v", err))
	}

	// 2. 推送到本地订阅者（发布订阅模式）
	dp.pushToSubscribers(msg.Channel, msg.Data)

	return protocol.NewResponse(msg.ID, map[string]string{
		"status":  "published",
		"channel": msg.Channel,
	})
}

// handleSubscribe 订阅频道
func (dp *DataProcessor) handleSubscribe(connID string, conn protocol.ConnInterface, msg *protocol.Message) *protocol.Response {
	if msg.Channel == "" {
		return protocol.NewErrorResponse(msg.ID, "channel is required")
	}

	dp.subscribersMu.Lock()
	defer dp.subscribersMu.Unlock()

	if _, ok := dp.subscribers[msg.Channel]; !ok {
		dp.subscribers[msg.Channel] = make(map[string]protocol.ConnInterface)
	}

	dp.subscribers[msg.Channel][connID] = conn
	atomic.AddUint64(&dp.metrics.SubscribedCount, 1)

	log.Printf("[Processor] Connection %s subscribed to channel: %s", connID, msg.Channel)

	return protocol.NewResponse(msg.ID, map[string]string{
		"status":  "subscribed",
		"channel": msg.Channel,
	})
}

// pushToSubscribers 推送消息到订阅者
func (dp *DataProcessor) pushToSubscribers(channel string, data interface{}) {
	dp.subscribersMu.RLock()
	subs, ok := dp.subscribers[channel]
	dp.subscribersMu.RUnlock()

	if !ok || len(subs) == 0 {
		return
	}

	resp := &protocol.Response{
		ID:     "pubsub",
		Status: "ok",
		Data: map[string]interface{}{
			"type":    "message",
			"channel": channel,
			"data":    data,
		},
	}

	// 异步推送给所有订阅者
	for connID, conn := range subs {
		go func(id string, c protocol.ConnInterface) {
			if err := c.Send(resp); err != nil {
				log.Printf("[Processor] Push to subscriber %s failed: %v", id, err)
				// 移除失效订阅
				dp.removeSubscriber(channel, id)
			}
		}(connID, conn)
	}
}

// removeSubscriber 移除订阅者
func (dp *DataProcessor) removeSubscriber(channel, connID string) {
	dp.subscribersMu.Lock()
	defer dp.subscribersMu.Unlock()

	if subs, ok := dp.subscribers[channel]; ok {
		delete(subs, connID)
		if len(subs) == 0 {
			delete(dp.subscribers, channel)
		}
	}
}

// handleGetMetrics 获取指标
func (dp *DataProcessor) handleGetMetrics(msg *protocol.Message) *protocol.Response {
	return protocol.NewResponse(msg.ID, dp.GetMetrics())
}

// handleGetConfig 获取配置
func (dp *DataProcessor) handleGetConfig(msg *protocol.Message) *protocol.Response {
	return protocol.NewResponse(msg.ID, map[string]interface{}{
		"server": dp.config.Server,
		"app":    dp.config.App,
		"redis": map[string]interface{}{
			"Host": dp.config.Redis.Host,
			"Port": dp.config.Redis.Port,
			"DB":   dp.config.Redis.DB,
		},
	})
}

// updateMetrics 更新指标
func (dp *DataProcessor) updateMetrics(success bool, latency int64) {
	dp.metrics.mu.Lock()
	defer dp.metrics.mu.Unlock()

	if success {
		dp.metrics.ProcessedCount++
	} else {
		dp.metrics.FailedCount++
	}

	dp.metrics.TotalLatency += latency
	if dp.metrics.ProcessedCount > 0 {
		dp.metrics.AvgLatency = float64(dp.metrics.TotalLatency) / float64(dp.metrics.ProcessedCount)
	}
	dp.metrics.LastProcessTime = time.Now()
}

// GetMetrics 获取指标
func (dp *DataProcessor) GetMetrics() map[string]interface{} {
	dp.metrics.mu.RLock()
	defer dp.metrics.mu.RUnlock()

	return map[string]interface{}{
		"processed_count":  dp.metrics.ProcessedCount,
		"failed_count":     dp.metrics.FailedCount,
		"published_count":  atomic.LoadUint64(&dp.metrics.PublishedCount),
		"subscribed_count": atomic.LoadUint64(&dp.metrics.SubscribedCount),
		"avg_latency":      dp.metrics.AvgLatency,
		"last_process":     dp.metrics.LastProcessTime,
	}
}

// ResetMetrics 重置指标
func (dp *DataProcessor) ResetMetrics() {
	dp.metrics.mu.Lock()
	defer dp.metrics.mu.Unlock()

	dp.metrics.ProcessedCount = 0
	dp.metrics.FailedCount = 0
	dp.metrics.PublishedCount = 0
	dp.metrics.SubscribedCount = 0
	dp.metrics.TotalLatency = 0
	dp.metrics.AvgLatency = 0
}

// ProcessFromQueue 从Redis队列消费消息（用于异步处理）
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

		var data interface{}
		if err := json.Unmarshal([]byte(val), &data); err != nil {
			log.Printf("[Processor] Unmarshal error: %v", err)
			continue
		}

		// 推送到订阅者
		dp.pushToSubscribers(queueKey, data)
		processed++
	}

	return processed, nil
}
