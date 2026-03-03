// Package api HTTP管理服务器
package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"goserver/internal/cache"
	"goserver/internal/config"
	"goserver/internal/network"
	"goserver/internal/processor"
	"goserver/internal/protocol"
)

// HTTPServer HTTP管理服务器
type HTTPServer struct {
	router    *gin.Engine
	server    *http.Server
	config    *config.Config
	redis     *cache.RedisClient
	tcpServer *network.TCPServer
	processor *processor.LogProcessor
}

// NewHTTPServer 创建HTTP服务器
func NewHTTPServer(cfg *config.Config, redis *cache.RedisClient, tcpServer *network.TCPServer, proc *processor.LogProcessor) *HTTPServer {
	if cfg.App.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// 静态文件
	router.Static("/ui", "./web/public")
	router.StaticFile("/", "./web/public/index.html")

	s := &HTTPServer{
		router:    router,
		config:    cfg,
		redis:     redis,
		tcpServer: tcpServer,
		processor: proc,
	}

	s.setupRoutes()
	return s
}

// setupRoutes 设置路由
func (s *HTTPServer) setupRoutes() {
	// 健康检查
	s.router.GET("/health", s.healthCheck)

	// API路由组
	api := s.router.Group("/api")
	{
		// 系统配置
		api.GET("/config", s.getConfig)
		
		// 指标统计
		api.GET("/metrics", s.getMetrics)
		api.POST("/metrics/reset", s.resetMetrics)

		// 日志管理
		api.GET("/logs", s.queryLogs)           // 查询日志
		api.GET("/logs/recent", s.getRecentLogs) // 获取最近日志
		api.GET("/logs/stats", s.getLogStats)   // 日志统计
		api.DELETE("/logs", s.clearLogs)        // 清空日志
		
		// 日志级别分布
		api.GET("/logs/levels", s.getLevelDistribution)
		
		// 系统连接
		api.GET("/connections", s.getConnections)
	}
}

// Start 启动HTTP服务器
func (s *HTTPServer) Start(port string) error {
	s.server = &http.Server{
		Addr:    ":" + port,
		Handler: s.router,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[HTTP Server] Error: %v", err)
		}
	}()

	log.Printf("[HTTP Server] Started on port %s", port)
	return nil
}

// Stop 停止HTTP服务器
func (s *HTTPServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// healthCheck 健康检查
func (s *HTTPServer) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"app":       s.config.App.Name,
		"version":   s.config.App.Version,
		"tcp_port":  s.config.Server.Port,
		"http_port": s.config.Server.Port,
	})
}

// getConfig 获取配置
func (s *HTTPServer) getConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"server": s.config.Server,
		"app":    s.config.App,
	})
}

// getMetrics 获取指标
func (s *HTTPServer) getMetrics(c *gin.Context) {
	tcpStats := s.tcpServer.GetStats()
	processorMetrics := s.processor.GetMetrics()

	c.JSON(http.StatusOK, gin.H{
		"system": map[string]interface{}{
			"active_connections": tcpStats.ActiveConnections,
			"total_connections":  tcpStats.TotalConnections,
		},
		"logs": processorMetrics,
		"timestamp": time.Now().Unix(),
	})
}

// resetMetrics 重置指标
func (s *HTTPServer) resetMetrics(c *gin.Context) {
	s.processor.ResetMetrics()
	c.JSON(http.StatusOK, gin.H{"message": "metrics reset"})
}

// queryLogs 查询日志
func (s *HTTPServer) queryLogs(c *gin.Context) {
	filters := &protocol.LogFilter{
		Level:    c.Query("level"),
		Source:   c.Query("source"),
		Limit:    parseInt(c.DefaultQuery("limit", "100")),
	}
	
	// 时间范围
	if start := c.Query("start_time"); start != "" {
		filters.StartTime = parseInt64(start)
	}
	if end := c.Query("end_time"); end != "" {
		filters.EndTime = parseInt64(end)
	}
	
	msg := &protocol.Message{
		ID:      "http-query",
		Cmd:     protocol.CmdQuery,
		Filters: filters,
	}
	
	resp := s.processor.Handle("http", nil, msg)
	c.JSON(http.StatusOK, resp)
}

// getRecentLogs 获取最近日志
func (s *HTTPServer) getRecentLogs(c *gin.Context) {
	limit := parseInt(c.DefaultQuery("limit", "50"))
	if limit > 500 {
		limit = 500
	}
	
	msg := &protocol.Message{
		ID:  "http-recent",
		Cmd: protocol.CmdQuery,
		Filters: &protocol.LogFilter{
			Limit: limit,
		},
	}
	
	resp := s.processor.Handle("http", nil, msg)
	c.JSON(http.StatusOK, resp)
}

// getLogStats 获取日志统计
func (s *HTTPServer) getLogStats(c *gin.Context) {
	ctx := context.Background()
	dateStr := time.Now().Format("2006-01-02")
	
	// 获取各级别日志数量
	levels := []string{"debug", "info", "warn", "error", "fatal"}
	stats := make(map[string]int64)
	
	for _, level := range levels {
		key := fmt.Sprintf("stats:logs:%s:%s", dateStr, level)
		count, _ := s.redis.GetClient().Get(ctx, key).Int64()
		stats[level] = count
	}
	
	c.JSON(http.StatusOK, gin.H{
		"date":  dateStr,
		"stats": stats,
	})
}

// getLevelDistribution 获取日志级别分布
func (s *HTTPServer) getLevelDistribution(c *gin.Context) {
	days := parseInt(c.DefaultQuery("days", "7"))
	if days > 30 {
		days = 30
	}
	
	ctx := context.Background()
	levels := []string{"debug", "info", "warn", "error", "fatal"}
	distribution := make(map[string][]map[string]interface{})
	
	for _, level := range levels {
		var data []map[string]interface{}
		for i := 0; i < days; i++ {
			date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
			key := fmt.Sprintf("stats:logs:%s:%s", date, level)
			count, _ := s.redis.GetClient().Get(ctx, key).Int64()
			data = append(data, map[string]interface{}{
				"date":  date,
				"count": count,
			})
		}
		distribution[level] = data
	}
	
	c.JSON(http.StatusOK, gin.H{
		"distribution": distribution,
	})
}

// clearLogs 清空日志
func (s *HTTPServer) clearLogs(c *gin.Context) {
	ctx := context.Background()
	
	// 获取所有日志键
	pattern := "logs:*"
	keys, err := s.redis.Keys(ctx, pattern)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// 删除所有日志键
	if len(keys) > 0 {
		s.redis.Del(ctx, keys...)
	}
	
	// 删除统计键
	statsKeys, _ := s.redis.Keys(ctx, "stats:logs:*")
	if len(statsKeys) > 0 {
		s.redis.Del(ctx, statsKeys...)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "logs cleared",
		"deleted": len(keys) + len(statsKeys),
	})
}

// getConnections 获取连接信息
func (s *HTTPServer) getConnections(c *gin.Context) {
	connections := s.tcpServer.GetConnections()
	c.JSON(http.StatusOK, gin.H{
		"connections": connections,
		"count":       len(connections),
	})
}

// corsMiddleware 跨域中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// parseInt 解析整数
func parseInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

// parseInt64 解析int64
func parseInt64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}
