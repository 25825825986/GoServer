// Package network TCP连接管理
package network

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"goserver/internal/protocol"
)

// ConnState 连接状态
type ConnState int32

const (
	ConnStateActive ConnState = iota
	ConnStateClosing
	ConnStateClosed
)

// Connection 包装TCP连接
type Connection struct {
	ID       string
	conn     net.Conn
	server   *TCPServer
	state    int32

	// 心跳相关
	lastActiveTime int64 // 最后活跃时间戳
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration

	// 读写
	reader *bufio.Reader
	writer *bufio.Writer
	writeMu sync.Mutex

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewConnection 创建新连接
func NewConnection(id string, conn net.Conn, server *TCPServer) *Connection {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Connection{
		ID:                id,
		conn:              conn,
		server:            server,
		state:             int32(ConnStateActive),
		lastActiveTime:    time.Now().Unix(),
		heartbeatInterval: 30 * time.Second,
		heartbeatTimeout:  90 * time.Second,
		reader:            bufio.NewReader(conn),
		writer:            bufio.NewWriter(conn),
		ctx:               ctx,
		cancel:            cancel,
	}
	return c
}

// Start 启动连接处理
func (c *Connection) Start() {
	c.wg.Add(2)
	go c.readLoop()
	go c.heartbeatLoop()
}

// readLoop 读取循环
func (c *Connection) readLoop() {
	defer c.wg.Done()
	defer c.Close()

	decoder := json.NewDecoder(c.reader)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		// 设置读取超时
		c.conn.SetReadDeadline(time.Now().Add(c.heartbeatTimeout))

		var msg protocol.Message
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				log.Printf("[Conn-%s] Client disconnected", c.ID)
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 超时继续等待
				continue
			}
			// 可能是非JSON数据（如HTTP请求），记录一次后关闭连接
			log.Printf("[Conn-%s] Invalid data format (maybe HTTP request?), closing connection", c.ID)
			return
		}

		// 更新活跃时间
		atomic.StoreInt64(&c.lastActiveTime, time.Now().Unix())

		// 处理心跳
		if msg.Cmd == protocol.CmdPing {
			c.Send(&protocol.Response{
				ID:     msg.ID,
				Status: "ok",
				Data:   map[string]string{"type": "pong"},
			})
			continue
		}

		// 提交到工作池处理
		c.server.workerPool.SubmitTask(&protocol.Task{
			ConnID:     c.ID,
			Message:    &msg,
			Conn:       c,
			SubmitTime: time.Now(),
		})
	}
}

// heartbeatLoop 心跳检测循环
func (c *Connection) heartbeatLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			lastActive := atomic.LoadInt64(&c.lastActiveTime)
			if time.Now().Unix()-lastActive > int64(c.heartbeatTimeout.Seconds()) {
				log.Printf("[Conn-%s] Heartbeat timeout, closing", c.ID)
				c.Close()
				return
			}
		}
	}
}

// Send 发送响应
func (c *Connection) Send(resp *protocol.Response) error {
	if c.IsClosed() {
		return fmt.Errorf("connection closed")
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	// 设置写入超时
	c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	data = append(data, '\n')

	if _, err := c.writer.Write(data); err != nil {
		return err
	}

	return c.writer.Flush()
}

// Close 关闭连接
func (c *Connection) Close() error {
	if !atomic.CompareAndSwapInt32(&c.state, int32(ConnStateActive), int32(ConnStateClosing)) {
		return nil
	}

	c.cancel()
	c.wg.Wait()

	if c.conn != nil {
		c.conn.Close()
	}

	atomic.StoreInt32(&c.state, int32(ConnStateClosed))
	
	// 从服务器移除
	c.server.removeConnection(c.ID)
	
	log.Printf("[Conn-%s] Closed", c.ID)
	return nil
}

// IsClosed 检查连接是否已关闭
func (c *Connection) IsClosed() bool {
	return atomic.LoadInt32(&c.state) == int32(ConnStateClosed)
}

// GetRemoteAddr 获取远程地址
func (c *Connection) GetRemoteAddr() string {
	if c.conn != nil {
		return c.conn.RemoteAddr().String()
	}
	return ""
}

// GetID 获取连接ID（实现protocol.ConnInterface）
func (c *Connection) GetID() string {
	return c.ID
}
