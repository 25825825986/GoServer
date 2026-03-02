package main

import (
	"log"
	"os"

	"goserver/internal/api"
	"goserver/internal/cache"
	"goserver/internal/config"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化Redis连接
	redisClient := cache.NewRedisClient(cfg)
	defer redisClient.Close()

	// 验证Redis连接
	if err := redisClient.Ping(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("[OK] Redis connection established")

	// 初始化API服务
	server := api.NewServer(cfg, redisClient)

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.Server.Port
	}

	log.Printf("Starting server on port %s...", port)
	if err := server.Start(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
