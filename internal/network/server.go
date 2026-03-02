// Package network TCP服务器实现
package network

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"

	"goserver/internal/config"
	"goserver/internal/pool"
	"goserver/internal/protocol"
)

// TCPServer TCP服务器
type TCPServer struct {
	config     *config.Config
	handler    protocol.Handler
	workerPool *pool.WorkerPool

	// 网络
	listener net.Listener
	address  string

	// 连接管理
	connections   map[string]*Connection
	connectionsMu sync.RWMutex
	connIDCounter uint64

	// 状态
	running int32
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	// 统计
	stats *ServerStats
}

// ServerStats 服务器统计
type ServerStats struct {
	TotalConnections   uint64
	ActiveConnections  uint64
	TotalMessages      uint64
	TotalBytesReceived uint64
	TotalBytesSent     uint64
}

// NewTCPServer 创建TCP服务器
func NewTCPServer(cfg *config.Config, handler protocol.Handler) *TCPServer {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &TCPServer{
		config:      cfg,
		handler:     handler,
		workerPool:  pool.NewWorkerPool(cfg.App.Workers, cfg.App.QueueSize),
		address:     fmt.Sprintf(":%s", cfg.Server.Port),
		connections: make(map[string]*Connection),
		ctx:         ctx,
		cancel:      cancel,
		stats:       &ServerStats{},
	}
}

// Start 启动服务器
func (s *TCPServer) Start() error {
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		return fmt.Errorf("server already running")
	}

	// 设置处理器并启动工作池
	s.workerPool.SetHandler(s.handler)
	s.workerPool.Start()

	// 监听端口
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.address, err)
	}
	s.listener = listener

	log.Printf("[TCP Server] Started on %s", s.address)
	log.Printf("[TCP Server] Max workers: %d, Queue size: %d", s.config.App.Workers, s.config.App.QueueSize)

	// 接受连接
	s.wg.Add(1)
	go s.acceptLoop()

	return nil
}

// acceptLoop 接受连接循环
func (s *TCPServer) acceptLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			if s.ctx.Err() != nil {
				return
			}
			log.Printf("[TCP Server] Accept error: %v", err)
			continue
		}

		// 检查连接数限制（支持1万并发）
		if s.GetActiveConnections() >= 10000 {
			log.Printf("[TCP Server] Max connections reached, rejecting %s", conn.RemoteAddr())
			conn.Close()
			continue
		}

		s.handleConnection(conn)
	}
}

// handleConnection 处理新连接
func (s *TCPServer) handleConnection(conn net.Conn) {
	connID := fmt.Sprintf("conn-%d", atomic.AddUint64(&s.connIDCounter, 1))
	
	connection := NewConnection(connID, conn, s)
	
	s.connectionsMu.Lock()
	s.connections[connID] = connection
	s.connectionsMu.Unlock()

	atomic.AddUint64(&s.stats.TotalConnections, 1)
	atomic.AddUint64(&s.stats.ActiveConnections, 1)

	log.Printf("[TCP Server] New connection: %s from %s", connID, conn.RemoteAddr())

	// 启动连接处理
	connection.Start()
}

// removeConnection 移除连接
func (s *TCPServer) removeConnection(connID string) {
	s.connectionsMu.Lock()
	delete(s.connections, connID)
	s.connectionsMu.Unlock()

	atomic.AddUint64(&s.stats.ActiveConnections, ^uint64(0)) // -1
}

// Stop 停止服务器
func (s *TCPServer) Stop() error {
	if !atomic.CompareAndSwapInt32(&s.running, 1, 0) {
		return nil
	}

	log.Printf("[TCP Server] Stopping...")

	// 停止接受新连接
	s.cancel()
	if s.listener != nil {
		s.listener.Close()
	}

	// 关闭所有连接
	s.connectionsMu.Lock()
	conns := make([]*Connection, 0, len(s.connections))
	for _, conn := range s.connections {
		conns = append(conns, conn)
	}
	s.connectionsMu.Unlock()

	for _, conn := range conns {
		conn.Close()
	}

	// 等待所有连接关闭
	s.wg.Wait()

	// 停止工作池
	s.workerPool.Stop()

	log.Printf("[TCP Server] Stopped")
	return nil
}

// GetActiveConnections 获取活跃连接数
func (s *TCPServer) GetActiveConnections() uint64 {
	return atomic.LoadUint64(&s.stats.ActiveConnections)
}

// GetStats 获取统计信息
func (s *TCPServer) GetStats() ServerStats {
	return ServerStats{
		TotalConnections:   atomic.LoadUint64(&s.stats.TotalConnections),
		ActiveConnections:  atomic.LoadUint64(&s.stats.ActiveConnections),
		TotalMessages:      atomic.LoadUint64(&s.stats.TotalMessages),
		TotalBytesReceived: atomic.LoadUint64(&s.stats.TotalBytesReceived),
		TotalBytesSent:     atomic.LoadUint64(&s.stats.TotalBytesSent),
	}
}

// GetWorkerPoolStats 获取工作池统计
func (s *TCPServer) GetWorkerPoolStats() map[string]interface{} {
	stats := s.workerPool.GetStats()
	return map[string]interface{}{
		"tasks_submitted": stats.TasksSubmitted,
		"tasks_processed": stats.TasksProcessed,
		"tasks_failed":    stats.TasksFailed,
		"avg_latency":     stats.AvgLatency,
		"queue_length":    stats.QueueLength,
	}
}

// Broadcast 广播消息到所有连接（用于发布订阅）
func (s *TCPServer) Broadcast(channel string, data interface{}) {
	s.connectionsMu.RLock()
	conns := make([]*Connection, 0, len(s.connections))
	for _, conn := range s.connections {
		conns = append(conns, conn)
	}
	s.connectionsMu.RUnlock()

	resp := &protocol.Response{
		ID:     "broadcast",
		Status: "ok",
		Data: map[string]interface{}{
			"channel": channel,
			"data":    data,
		},
	}

	for _, conn := range conns {
		go conn.Send(resp)
	}
}
