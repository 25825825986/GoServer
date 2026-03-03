// Package processor 日志处理逻辑
package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"goserver/internal/cache"
	"goserver/internal/config"
	"goserver/internal/protocol"
)

// LogProcessor 日志处理器
type LogProcessor struct {
	redis      *cache.RedisClient
	config     *config.Config
	
	// 指标统计
	metrics *ProcessorMetrics
}

// ProcessorMetrics 处理器指标
type ProcessorMetrics struct {
	mu              sync.RWMutex
	LogsReceived    uint64    // 接收的日志数
	LogsProcessed   uint64    // 处理成功的日志数
	LogsFailed      uint64    // 处理失败的日志数
	TotalLatency    int64     // 总延迟（毫秒）
	AvgLatency      float64   // 平均延迟
	LastProcessTime time.Time // 最后处理时间
}

// NewLogProcessor 创建日志处理器
func NewLogProcessor(redis *cache.RedisClient, cfg *config.Config) *LogProcessor {
	return &LogProcessor{
		redis:   redis,
		config:  cfg,
		metrics: &ProcessorMetrics{},
	}
}

// Handle 处理消息（实现protocol.Handler接口）
func (lp *LogProcessor) Handle(connID string, conn protocol.ConnInterface, msg *protocol.Message) *protocol.Response {
	start := time.Now()
	
	var resp *protocol.Response
	
	switch msg.Cmd {
	case protocol.CmdLog:
		resp = lp.handleSingleLog(msg)
	case protocol.CmdBatch:
		resp = lp.handleBatchLogs(msg)
	case protocol.CmdQuery:
		resp = lp.handleQuery(msg)
	case protocol.CmdPing:
		resp = &protocol.Response{ID: msg.ID, Status: "ok", Data: map[string]string{"type": "pong"}}
	default:
		resp = protocol.NewErrorResponse(msg.ID, fmt.Sprintf("unknown command: %s", msg.Cmd))
	}
	
	// 更新指标
	latency := time.Since(start).Milliseconds()
	resp.Latency = latency
	lp.updateMetrics(resp.Status == "ok", latency)
	
	return resp
}

// handleSingleLog 处理单条日志
func (lp *LogProcessor) handleSingleLog(msg *protocol.Message) *protocol.Response {
	// 解析日志数据
	logData, err := lp.parseLogEntry(msg.Data)
	if err != nil {
		return protocol.NewErrorResponse(msg.ID, fmt.Sprintf("parse log failed: %v", err))
	}
	
	// 存储到Redis
	if err := lp.storeLog(logData); err != nil {
		return protocol.NewErrorResponse(msg.ID, fmt.Sprintf("store log failed: %v", err))
	}
	
	return protocol.NewResponse(msg.ID, map[string]string{
		"status": "stored",
		"level":  string(logData.Level),
	})
}

// handleBatchLogs 批量处理日志
func (lp *LogProcessor) handleBatchLogs(msg *protocol.Message) *protocol.Response {
	batchData, ok := msg.Data.([]interface{})
	if !ok {
		return protocol.NewErrorResponse(msg.ID, "invalid batch data format")
	}
	
	successCount := 0
	for _, item := range batchData {
		logData, err := lp.parseLogEntry(item)
		if err != nil {
			continue
		}
		if err := lp.storeLog(logData); err != nil {
			continue
		}
		successCount++
	}
	
	return &protocol.Response{
		ID:     msg.ID,
		Status: "ok",
		Data: map[string]interface{}{
			"total":   len(batchData),
			"success": successCount,
			"failed":  len(batchData) - successCount,
		},
		Count: successCount,
	}
}

// handleQuery 处理日志查询
func (lp *LogProcessor) handleQuery(msg *protocol.Message) *protocol.Response {
	filters := msg.Filters
	if filters == nil {
		filters = &protocol.LogFilter{Limit: 100}
	}
	if filters.Limit == 0 || filters.Limit > 1000 {
		filters.Limit = 100
	}
	
	logs, err := lp.queryLogs(filters)
	if err != nil {
		return protocol.NewErrorResponse(msg.ID, fmt.Sprintf("query failed: %v", err))
	}
	
	return &protocol.Response{
		ID:     msg.ID,
		Status: "ok",
		Data:   logs,
		Count:  len(logs),
	}
}

// parseLogEntry 解析日志条目
func (lp *LogProcessor) parseLogEntry(data interface{}) (*protocol.LogEntry, error) {
	if data == nil {
		return nil, fmt.Errorf("log data is nil")
	}
	
	var entry protocol.LogEntry
	
	switch v := data.(type) {
	case map[string]interface{}:
		// 填充时间戳
		if ts, ok := v["timestamp"].(float64); ok {
			entry.Timestamp = int64(ts)
		} else {
			entry.Timestamp = time.Now().UnixMilli()
		}
		
		// 日志级别
		if level, ok := v["level"].(string); ok {
			entry.Level = protocol.LogLevel(level)
		} else {
			entry.Level = protocol.LevelInfo
		}
		
		// 来源
		if source, ok := v["source"].(string); ok {
			entry.Source = source
		}
		
		// 消息内容
		if msg, ok := v["message"].(string); ok {
			entry.Message = msg
		} else {
			return nil, fmt.Errorf("log message is required")
		}
		
		// 标签
		if tags, ok := v["tags"].([]interface{}); ok {
			for _, t := range tags {
				if tag, ok := t.(string); ok {
					entry.Tags = append(entry.Tags, tag)
				}
			}
		}
		
		// 元数据
		if meta, ok := v["metadata"]; ok {
			entry.Metadata = meta
		}
		
	case string:
		// 简单字符串格式，作为消息内容
		entry = protocol.LogEntry{
			Timestamp: time.Now().UnixMilli(),
			Level:     protocol.LevelInfo,
			Message:   v,
		}
		
	default:
		// 尝试JSON序列化再反序列化
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal log data: %v", err)
		}
		if err := json.Unmarshal(jsonData, &entry); err != nil {
			return nil, fmt.Errorf("failed to unmarshal log data: %v", err)
		}
	}
	
	// 默认填充
	if entry.Timestamp == 0 {
		entry.Timestamp = time.Now().UnixMilli()
	}
	if entry.Level == "" {
		entry.Level = protocol.LevelInfo
	}
	
	return &entry, nil
}

// storeLog 存储日志到Redis
func (lp *LogProcessor) storeLog(entry *protocol.LogEntry) error {
	ctx := context.Background()
	
	// 生成存储键：按日期和级别分类
	dateStr := time.UnixMilli(entry.Timestamp).Format("2006-01-02")
	
	// 1. 存储到全局日志列表（按时间排序）
	logJSON, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	
	// 使用Redis Stream或List存储
	key := fmt.Sprintf("logs:%s", dateStr)
	if _, err := lp.redis.Lpush(ctx, key, string(logJSON)); err != nil {
		return err
	}
	
	// 2. 按级别索引
	levelKey := fmt.Sprintf("logs:%s:%s", dateStr, entry.Level)
	lp.redis.Lpush(ctx, levelKey, string(logJSON))
	
	// 3. 按来源索引
	if entry.Source != "" {
		sourceKey := fmt.Sprintf("logs:source:%s", entry.Source)
		lp.redis.Lpush(ctx, sourceKey, string(logJSON))
	}
	
	// 4. 设置过期时间（保留7天）
	lp.redis.Expire(ctx, key, 7*24*time.Hour)
	
	// 5. 更新统计计数器
	lp.redis.Incr(ctx, fmt.Sprintf("stats:logs:%s:%s", dateStr, entry.Level))
	
	return nil
}

// queryLogs 查询日志
func (lp *LogProcessor) queryLogs(filters *protocol.LogFilter) ([]*protocol.LogEntry, error) {
	ctx := context.Background()
	var results []*protocol.LogEntry
	
	// 确定查询的键
	var key string
	dateStr := time.Now().Format("2006-01-02")
	
	if filters.Level != "" {
		key = fmt.Sprintf("logs:%s:%s", dateStr, filters.Level)
	} else if filters.Source != "" {
		key = fmt.Sprintf("logs:source:%s", filters.Source)
	} else {
		key = fmt.Sprintf("logs:%s", dateStr)
	}
	
	// 从Redis获取日志
	logs, err := lp.redis.LRange(ctx, key, 0, int64(filters.Limit-1))
	if err != nil {
		return nil, err
	}
	
	for _, logJSON := range logs {
		var entry protocol.LogEntry
		if err := json.Unmarshal([]byte(logJSON), &entry); err != nil {
			continue
		}
		
		// 时间过滤
		if filters.StartTime > 0 && entry.Timestamp < filters.StartTime {
			continue
		}
		if filters.EndTime > 0 && entry.Timestamp > filters.EndTime {
			continue
		}
		
		// 关键词过滤
		if len(filters.Keywords) > 0 {
			matched := false
			for _, kw := range filters.Keywords {
				if contains(entry.Message, kw) || contains(entry.Source, kw) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		
		results = append(results, &entry)
	}
	
	return results, nil
}

// updateMetrics 更新指标
func (lp *LogProcessor) updateMetrics(success bool, latency int64) {
	lp.metrics.mu.Lock()
	defer lp.metrics.mu.Unlock()
	
	atomic.AddUint64(&lp.metrics.LogsReceived, 1)
	
	if success {
		atomic.AddUint64(&lp.metrics.LogsProcessed, 1)
	} else {
		atomic.AddUint64(&lp.metrics.LogsFailed, 1)
	}
	
	lp.metrics.TotalLatency += latency
	processed := atomic.LoadUint64(&lp.metrics.LogsProcessed)
	if processed > 0 {
		lp.metrics.AvgLatency = float64(lp.metrics.TotalLatency) / float64(processed)
	}
	lp.metrics.LastProcessTime = time.Now()
}

// GetMetrics 获取指标
func (lp *LogProcessor) GetMetrics() map[string]interface{} {
	lp.metrics.mu.RLock()
	defer lp.metrics.mu.RUnlock()
	
	return map[string]interface{}{
		"logs_received":   atomic.LoadUint64(&lp.metrics.LogsReceived),
		"logs_processed":  atomic.LoadUint64(&lp.metrics.LogsProcessed),
		"logs_failed":     atomic.LoadUint64(&lp.metrics.LogsFailed),
		"avg_latency_ms":  lp.metrics.AvgLatency,
		"last_process":    lp.metrics.LastProcessTime,
	}
}

// ResetMetrics 重置指标
func (lp *LogProcessor) ResetMetrics() {
	lp.metrics.mu.Lock()
	defer lp.metrics.mu.Unlock()
	
	lp.metrics.LogsReceived = 0
	lp.metrics.LogsProcessed = 0
	lp.metrics.LogsFailed = 0
	lp.metrics.TotalLatency = 0
	lp.metrics.AvgLatency = 0
}

// GetStats 获取统计（兼容旧接口）
func (lp *LogProcessor) GetStats() map[string]interface{} {
	return lp.GetMetrics()
}

// contains 字符串包含检查
func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
