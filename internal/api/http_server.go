// Package api HTTP管理服务器
package api

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"goserver/internal/cache"
	"goserver/internal/config"
	"goserver/internal/network"
	"goserver/internal/processor"
)

// HTTPServer HTTP管理服务器
type HTTPServer struct {
	router    *gin.Engine
	server    *http.Server
	config    *config.Config
	redis     *cache.RedisClient
	tcpServer *network.TCPServer
	processor *processor.DataProcessor
}

// NewHTTPServer 创建HTTP服务器
func NewHTTPServer(cfg *config.Config, redis *cache.RedisClient, tcpServer *network.TCPServer, proc *processor.DataProcessor) *HTTPServer {
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
		// 配置
		api.GET("/config", s.getConfig)

		// 指标
		api.GET("/metrics", s.getMetrics)
		api.POST("/metrics/reset", s.resetMetrics)

		// 连接信息
		api.GET("/connections", s.getConnections)

		// Redis管理
		api.GET("/redis/keys", s.getRedisKeys)
		api.DELETE("/redis/key/:key", s.deleteRedisKey)
		api.GET("/redis/info", s.getRedisInfo)
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
		"redis": map[string]interface{}{
			"Host": s.config.Redis.Host,
			"Port": s.config.Redis.Port,
			"DB":   s.config.Redis.DB,
		},
	})
}

// getMetrics 获取指标
func (s *HTTPServer) getMetrics(c *gin.Context) {
	// 从TCP服务器获取统计
	tcpStats := s.tcpServer.GetStats()

	c.JSON(http.StatusOK, gin.H{
		"tcp": map[string]interface{}{
			"total_connections":  tcpStats.TotalConnections,
			"active_connections": tcpStats.ActiveConnections,
			"total_messages":     tcpStats.TotalMessages,
		},
		"worker_pool": s.tcpServer.GetWorkerPoolStats(),
		"timestamp":   time.Now().Unix(),
	})
}

// resetMetrics 重置指标
func (s *HTTPServer) resetMetrics(c *gin.Context) {
	// 重置Processor统计
	if s.processor != nil {
		s.processor.ResetMetrics()
	}
	c.JSON(http.StatusOK, gin.H{"message": "metrics reset"})
}

// getConnections 获取连接信息
func (s *HTTPServer) getConnections(c *gin.Context) {
	stats := s.tcpServer.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"active_connections": stats.ActiveConnections,
		"total_connections":  stats.TotalConnections,
	})
}

// getRedisKeys 获取Redis键
func (s *HTTPServer) getRedisKeys(c *gin.Context) {
	ctx := c.Request.Context()
	pattern := c.DefaultQuery("pattern", "*")

	keys, err := s.redis.Keys(ctx, pattern)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(keys),
		"keys":  keys,
	})
}

// deleteRedisKey 删除Redis键
func (s *HTTPServer) deleteRedisKey(c *gin.Context) {
	key := c.Param("key")
	ctx := c.Request.Context()

	count, err := s.redis.Del(ctx, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "key deleted",
		"deleted": count,
	})
}

// getRedisInfo 获取Redis信息
func (s *HTTPServer) getRedisInfo(c *gin.Context) {
	ctx := c.Request.Context()
	cmd := s.redis.GetClient().Info(ctx, "server")

	c.JSON(http.StatusOK, gin.H{
		"info": cmd.Val(),
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

// WebConfig Web配置页面数据
type WebConfig struct {
	Server config.ServerConfig `json:"server"`
	App    config.AppConfig    `json:"app"`
	Redis  struct {
		Host string `json:"host"`
		Port string `json:"port"`
		DB   int    `json:"db"`
	} `json:"redis"`
}

// GetWebConfig 获取Web配置
func (s *HTTPServer) GetWebConfig() map[string]interface{} {
	return map[string]interface{}{
		"server": s.config.Server,
		"app":    s.config.App,
		"redis": map[string]interface{}{
			"Host": s.config.Redis.Host,
			"Port": s.config.Redis.Port,
			"DB":   s.config.Redis.DB,
		},
	}
}

// GetStats 获取统计信息
func (s *HTTPServer) GetStats() map[string]interface{} {
	tcpStats := s.tcpServer.GetStats()
	return map[string]interface{}{
		"tcp_connections": tcpStats.ActiveConnections,
	}
}

// convertIntQuery 转换整数查询参数
func convertIntQuery(c *gin.Context, key string, defaultVal int) int {
	val := c.DefaultQuery(key, strconv.Itoa(defaultVal))
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return intVal
}
