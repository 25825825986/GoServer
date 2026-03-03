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
	log.Println("[START] High Concurrency Data Processing System")

	// 1. 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("[FATAL] Failed to load config: %v", err)
	}
	// 设置端口：TCP用8080，HTTP用8081（避免冲突）
	tcpPort := cfg.Server.Port
	if tcpPort == "" {
		tcpPort = "8080"
	}
	httpPort := "8081"
	
	log.Printf("[CONFIG] Loaded: TCP Port=%s, HTTP Port=%s, Workers=%d, QueueSize=%d",
		tcpPort, httpPort, cfg.App.Workers, cfg.App.QueueSize)

	// 2. 初始化Redis连接
	redisClient := cache.NewRedisClient(cfg)
	defer redisClient.Close()

	if err := redisClient.Ping(); err != nil {
		log.Fatalf("[FATAL] Failed to connect to Redis: %v", err)
	}
	log.Println("[OK] Redis connection established")

	// 3. 创建业务处理器
	handler := processor.NewDataProcessor(redisClient, cfg)

	// 4. 创建TCP服务器（数据处理）
	tcpServer := network.NewTCPServer(cfg, handler)

	// 5. 启动TCP服务器
	if err := tcpServer.Start(); err != nil {
		log.Fatalf("[FATAL] Failed to start TCP server: %v", err)
	}
	log.Printf("[OK] TCP Server started on port %s", tcpPort)

	// 6. 创建并启动HTTP管理服务器（传入processor参数）
	httpServer := api.NewHTTPServer(cfg, redisClient, tcpServer, handler)
	if err := httpServer.Start(httpPort); err != nil {
		log.Fatalf("[FATAL] Failed to start HTTP server: %v", err)
	}
	log.Printf("[OK] HTTP Server started on port %s", httpPort)

	log.Println("")
	log.Println("===============================================")
	log.Println("  Server started successfully!")
	log.Println("===============================================")
	log.Printf("  TCP Data Server:  port %s (for data processing)", tcpPort)
	log.Printf("  HTTP Web UI:      port %s (for management)", httpPort)
	log.Printf("  Web Dashboard:    http://localhost:%s", httpPort)
	log.Println("===============================================")
	log.Println("")
	log.Println("[INFO] Max concurrency: 10000 TCP connections")
	log.Println("[INFO] Target QPS: 5000+")
	log.Println("")

	// 7. 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	log.Println("[WAIT] Press Ctrl+C to stop server...")
	<-quit

	log.Println("")
	log.Println("[STOP] Shutting down servers...")
	
	// 停止HTTP服务器
	if err := httpServer.Stop(); err != nil {
		log.Printf("[ERROR] HTTP server stop error: %v", err)
	}
	
	// 停止TCP服务器
	if err := tcpServer.Stop(); err != nil {
		log.Printf("[ERROR] TCP server stop error: %v", err)
	}

	log.Println("[OK] All servers stopped")
}
