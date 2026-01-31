package api

import (
	"net/http"

	"goserver/internal/cache"
	"goserver/internal/config"
	"goserver/internal/processor"

	"github.com/gin-gonic/gin"
)

// Server API服务器
type Server struct {
	engine    *gin.Engine
	config    *config.Config
	redis     *cache.RedisClient
	processor *processor.DataProcessor
}

// NewServer 创建新的API服务器
func NewServer(cfg *config.Config, redis *cache.RedisClient) *Server {
	if cfg.App.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())

	proc := processor.NewDataProcessor(redis, cfg)

	s := &Server{
		engine:    engine,
		config:    cfg,
		redis:     redis,
		processor: proc,
	}

	s.setupRoutes()
	return s
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 健康检查
	s.engine.GET("/health", s.healthCheck)

	// 配置管理
	s.engine.GET("/api/config", s.getConfig)
	s.engine.POST("/api/config", s.updateConfig)

	// 数据处理
	s.engine.POST("/api/process", s.processData)
	s.engine.POST("/api/process/batch", s.processBatch)
	s.engine.POST("/api/process/queue", s.processQueue)

	// 指标查询
	s.engine.GET("/api/metrics", s.getMetrics)
	s.engine.POST("/api/metrics/reset", s.resetMetrics)

	// Redis管理
	s.engine.GET("/api/redis/info", s.getRedisInfo)
	s.engine.GET("/api/redis/keys", s.getRedisKeys)
	s.engine.DELETE("/api/redis/key/:key", s.deleteRedisKey)

	// Web UI（可视化配置）
	s.engine.Static("/ui", "./web/public")
	s.engine.GET("/", func(c *gin.Context) {
		c.File("./web/public/index.html")
	})
}

// healthCheck 健康检查
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"app":    s.config.App.Name,
		"version": s.config.App.Version,
	})
}

// getConfig 获取系统配置
func (s *Server) getConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"server": s.config.Server,
		"redis":  s.config.Redis,
		"app":    s.config.App,
	})
}

// updateConfig 更新系统配置
func (s *Server) updateConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "配置更新成功",
		"config":  s.config,
	})
}

// ProcessRequest 处理请求体
type ProcessRequest struct {
	Data interface{} `json:"data"`
}

// processData 处理单个数据
func (s *Server) processData(c *gin.Context) {
	var req ProcessRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.processor.Process(req.Data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "数据处理成功",
		"data":    req.Data,
	})
}

// BatchRequest 批处理请求体
type BatchRequest struct {
	Data []interface{} `json:"data"`
}

// processBatch 批量处理数据
func (s *Server) processBatch(c *gin.Context) {
	var req BatchRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	successCount, err := s.processor.ProcessBatch(req.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "批量处理完成",
		"total":         len(req.Data),
		"success_count": successCount,
		"failed_count":  len(req.Data) - successCount,
	})
}

// QueueRequest 队列处理请求体
type QueueRequest struct {
	QueueKey  string `json:"queue_key"`
	BatchSize int    `json:"batch_size"`
}

// processQueue 从队列处理数据
func (s *Server) processQueue(c *gin.Context) {
	var req QueueRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.BatchSize <= 0 {
		req.BatchSize = s.config.App.BatchSize
	}

	processed, err := s.processor.ProcessFromQueue(req.QueueKey, req.BatchSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "队列处理完成",
		"processed": processed,
	})
}

// getMetrics 获取处理指标
func (s *Server) getMetrics(c *gin.Context) {
	metrics := s.processor.GetMetrics()
	c.JSON(http.StatusOK, metrics)
}

// resetMetrics 重置指标
func (s *Server) resetMetrics(c *gin.Context) {
	s.processor.ResetMetrics()
	c.JSON(http.StatusOK, gin.H{"message": "指标已重置"})
}

// getRedisInfo 获取Redis信息
func (s *Server) getRedisInfo(c *gin.Context) {
	ctx := c.Request.Context()
	cmd := s.redis.GetClient().Info(ctx, "server")
	
	c.JSON(http.StatusOK, gin.H{
		"info": cmd.Val(),
	})
}

// getRedisKeys 获取Redis键
func (s *Server) getRedisKeys(c *gin.Context) {
	pattern := c.DefaultQuery("pattern", "*")
	ctx := c.Request.Context()
	
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
func (s *Server) deleteRedisKey(c *gin.Context) {
	key := c.Param("key")
	ctx := c.Request.Context()
	
	count, err := s.redis.Del(ctx, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "键已删除",
		"deleted": count,
	})
}

// Start 启动服务器
func (s *Server) Start(port string) error {
	return s.engine.Run(":" + port)
}
