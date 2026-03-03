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
	"goserver/internal/db"
	"goserver/internal/models"
	"goserver/internal/protocol"
)

// LogProcessor 日志处理器
type LogProcessor struct {
	mysql    *db.MySQLClient
	redis    *cache.RedisClient
	config   *config.Config
	
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
func NewLogProcessor(mysql *db.MySQLClient, redis *cache.RedisClient, cfg *config.Config) *LogProcessor {
	return &LogProcessor{
		mysql:   mysql,
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
	logEntry, err := lp.parseLogEntry(msg.Data)
	if err != nil {
		return protocol.NewErrorResponse(msg.ID, fmt.Sprintf("parse log failed: %v", err))
	}
	
	// 存储到 MySQL（主存储）
	if err := lp.storeLogToMySQL(logEntry); err != nil {
		return protocol.NewErrorResponse(msg.ID, fmt.Sprintf("store log to MySQL failed: %v", err))
	}
	
	// 更新 Redis 缓存和统计（辅助）
	lp.updateCache(logEntry)
	
	return protocol.NewResponse(msg.ID, map[string]string{
		"status": "stored",
		"level":  string(logEntry.Level),
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
		logEntry, err := lp.parseLogEntry(item)
		if err != nil {
			continue
		}
		if err := lp.storeLogToMySQL(logEntry); err != nil {
			continue
		}
		lp.updateCache(logEntry)
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
func (lp *LogProcessor) parseLogEntry(data interface{}) (*models.LogEntry, error) {
	if data == nil {
		return nil, fmt.Errorf("log data is nil")
	}
	
	var entry models.LogEntry
	
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
			entry.Level = models.LogLevel(level)
		} else {
			entry.Level = models.LevelInfo
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
			if metaMap, ok := meta.(map[string]interface{}); ok {
				entry.Metadata = metaMap
			}
		}
		
	case string:
		// 简单字符串格式，作为消息内容
		entry = models.LogEntry{
			Timestamp: time.Now().UnixMilli(),
			Level:     models.LevelInfo,
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
		entry.Level = models.LevelInfo
	}
	
	return &entry, nil
}

// storeLogToMySQL 存储日志到 MySQL
func (lp *LogProcessor) storeLogToMySQL(entry *models.LogEntry) error {
	return lp.mysql.GetDB().Create(entry).Error
}

// updateCache 更新 Redis 缓存
func (lp *LogProcessor) updateCache(entry *models.LogEntry) {
	ctx := context.Background()
	dateStr := time.UnixMilli(entry.Timestamp).Format("2006-01-02")
	
	// 1. 缓存最近日志（最近100条）
	logJSON, _ := json.Marshal(entry)
	lp.redis.Lpush(ctx, "logs:recent", string(logJSON))
	lp.redis.LTrim(ctx, "logs:recent", 0, 99) // 只保留100条
	
	// 2. 更新各级别计数（今日）
	lp.redis.Incr(ctx, fmt.Sprintf("stats:logs:%s:%s", dateStr, entry.Level))
	lp.redis.Expire(ctx, fmt.Sprintf("stats:logs:%s:%s", dateStr, entry.Level), 7*24*time.Hour)
	
	// 3. 更新来源统计
	if entry.Source != "" {
		lp.redis.Incr(ctx, fmt.Sprintf("stats:source:%s", entry.Source))
	}
}

// queryLogs 查询日志（从MySQL查询）
func (lp *LogProcessor) queryLogs(filters *protocol.LogFilter) ([]*models.LogEntry, error) {
	db := lp.mysql.GetDB()
	
	// 构建查询
	if filters.Level != "" {
		db = db.Where("level = ?", filters.Level)
	}
	if filters.Source != "" {
		db = db.Where("source = ?", filters.Source)
	}
	if filters.StartTime > 0 {
		db = db.Where("timestamp >= ?", filters.StartTime)
	}
	if filters.EndTime > 0 {
		db = db.Where("timestamp <= ?", filters.EndTime)
	}
	
	// 关键词搜索
	if len(filters.Keywords) > 0 && filters.Keywords[0] != "" {
		db = db.Where("message LIKE ?", "%"+filters.Keywords[0]+"%")
	}
	
	var logs []*models.LogEntry
	err := db.Order("timestamp DESC").Limit(filters.Limit).Find(&logs).Error
	return logs, err
}

// GetRecentLogs 获取最近日志（带缓存）
func (lp *LogProcessor) GetRecentLogs(limit int) ([]*models.LogEntry, error) {
	// 先从Redis缓存获取
	ctx := context.Background()
	cached, err := lp.redis.LRange(ctx, "logs:recent", 0, int64(limit-1))
	if err == nil && len(cached) > 0 {
		var logs []*models.LogEntry
		for _, item := range cached {
			var log models.LogEntry
			if err := json.Unmarshal([]byte(item), &log); err == nil {
				logs = append(logs, &log)
			}
		}
		if len(logs) > 0 {
			return logs, nil
		}
	}
	
	// 缓存未命中，从MySQL查询
	var logs []*models.LogEntry
	err = lp.mysql.GetDB().Order("timestamp DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

// GetLogStats 获取日志统计
func (lp *LogProcessor) GetLogStats(date string) map[string]int64 {
	ctx := context.Background()
	levels := []string{"debug", "info", "warn", "error", "fatal"}
	stats := make(map[string]int64)
	
	for _, level := range levels {
		key := fmt.Sprintf("stats:logs:%s:%s", date, level)
		count, _ := lp.redis.GetClient().Get(ctx, key).Int64()
		stats[level] = count
	}
	
	return stats
}

// ClearAllLogs 清空所有日志
func (lp *LogProcessor) ClearAllLogs() error {
	// 清空MySQL
	if err := lp.mysql.GetDB().Exec("TRUNCATE TABLE logs").Error; err != nil {
		return err
	}
	
	// 清空Redis缓存
	ctx := context.Background()
	lp.redis.Del(ctx, "logs:recent")
	
	return nil
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
