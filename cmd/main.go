package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"goserver/internal/api"
	"goserver/internal/cache"
	"goserver/internal/config"
	"goserver/internal/network"
	"goserver/internal/processor"
)

func main() {
	log.Println("[START] Log Processing System")

	// 1. 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("[FATAL] Failed to load config: %v", err)
	}
	
	// 设置端口
	tcpPort := cfg.Server.Port
	if tcpPort == "" {
		tcpPort = "8080"
	}
	httpPort := "8081"
	
	log.Printf("[CONFIG] TCP Port=%s, HTTP Port=%s, Workers=%d, QueueSize=%d",
		tcpPort, httpPort, cfg.App.Workers, cfg.App.QueueSize)

	// 2. 初始化Redis连接
	redisClient := cache.NewRedisClient(cfg)
	defer redisClient.Close()

	if err := redisClient.Ping(); err != nil {
		log.Fatalf("[FATAL] Failed to connect to Redis: %v", err)
	}
	log.Println("[OK] Redis connection established")

	// 3. 创建日志处理器
	handler := processor.NewLogProcessor(redisClient, cfg)

	// 4. 创建TCP服务器（接收日志）
	tcpServer := network.NewTCPServer(cfg, handler)

	// 5. 启动TCP服务器
	if err := tcpServer.Start(); err != nil {
		log.Fatalf("[FATAL] Failed to start TCP server: %v", err)
	}
	log.Printf("[OK] TCP Log Receiver started on port %s", tcpPort)

	// 6. 创建并启动HTTP管理服务器
	httpServer := api.NewHTTPServer(cfg, redisClient, tcpServer, handler)
	if err := httpServer.Start(httpPort); err != nil {
		log.Fatalf("[FATAL] Failed to start HTTP server: %v", err)
	}
	log.Printf("[OK] HTTP Web UI started on port %s", httpPort)

	log.Println("")
	log.Println("===============================================")
	log.Println("  Log Processing System Started!")
	log.Println("===============================================")
	log.Printf("  TCP Log Receiver: port %s", tcpPort)
	log.Printf("  HTTP Web UI:      http://localhost:%s", httpPort)
	log.Println("===============================================")
	log.Println("")
	log.Println("[INFO] Features:")
	log.Println("       - Real-time log collection via TCP")
	log.Println("       - High-concurrency log processing")
	log.Println("       - Redis-based log storage")
	log.Println("       - Web dashboard for log management")
	log.Println("")

	// 7. 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	log.Println("[WAIT] Press Ctrl+C to stop server...")
	<-quit

	log.Println("")
	log.Println("[STOP] Shutting down...")
	
	httpServer.Stop()
	tcpServer.Stop()

	log.Println("[OK] Server stopped")
}
