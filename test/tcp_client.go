// TCP客户端测试工具 - 日志实时处理系统
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"goserver/internal/protocol"
)

// Client TCP客户端
type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	mu     sync.Mutex
}

// NewClient 创建客户端
func NewClient(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}, nil
}

// Send 发送消息
func (c *Client) Send(msg *protocol.Message) (*protocol.Response, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 发送
	if _, err := c.writer.Write(append(data, '\n')); err != nil {
		return nil, err
	}
	if err := c.writer.Flush(); err != nil {
		return nil, err
	}

	// 接收响应
	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	var resp protocol.Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Close 关闭连接
func (c *Client) Close() {
	c.conn.Close()
}

// 单条日志测试
func singleLogTest(addr string) {
	client, err := NewClient(addr)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fmt.Println("=== Single Log Test ===")

	// 发送单条日志
	resp, err := client.Send(&protocol.Message{
		ID:  "1",
		Cmd: protocol.CmdLog,
		Data: map[string]interface{}{
			"timestamp": time.Now().UnixMilli(),
			"level":     "error",
			"source":    "api-server",
			"message":   "数据库连接失败",
			"tags":      []string{"db", "critical"},
			"metadata": map[string]interface{}{
				"error_code": 5001,
				"retry":      3,
			},
		},
	})
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	printResponse(resp)

	// 发送 info 级别日志
	resp, err = client.Send(&protocol.Message{
		ID:  "2",
		Cmd: protocol.CmdLog,
		Data: map[string]interface{}{
			"level":   "info",
			"source":  "app",
			"message": "服务启动成功",
			"tags":    []string{"startup"},
		},
	})
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	printResponse(resp)

	fmt.Println("=== Test Completed ===")
}

// 批量日志测试
func batchLogTest(addr string) {
	client, err := NewClient(addr)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fmt.Println("=== Batch Log Test ===")

	logs := []interface{}{
		map[string]interface{}{
			"timestamp": time.Now().UnixMilli(),
			"level":     "debug",
			"source":    "worker-1",
			"message":   "处理任务开始",
		},
		map[string]interface{}{
			"level":   "info",
			"source":  "worker-1",
			"message": "任务处理完成",
		},
		map[string]interface{}{
			"level":   "warn",
			"source":  "cache",
			"message": "缓存命中率低于阈值",
		},
		map[string]interface{}{
			"level":   "error",
			"source":  "payment",
			"message": "支付网关返回超时",
		},
		map[string]interface{}{
			"level":   "fatal",
			"source":  "db",
			"message": "主数据库连接断开",
		},
	}

	resp, err := client.Send(&protocol.Message{
		ID:   "batch-1",
		Cmd:  protocol.CmdBatch,
		Data: logs,
	})
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	printResponse(resp)
	fmt.Println("=== Test Completed ===")
}

// 日志查询测试
func queryLogTest(addr string) {
	client, err := NewClient(addr)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fmt.Println("=== Query Log Test ===")

	// 查询所有日志
	resp, err := client.Send(&protocol.Message{
		ID:  "query-1",
		Cmd: protocol.CmdQuery,
		Filters: &protocol.LogFilter{
			Limit: 10,
		},
	})
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	printResponse(resp)

	// 按级别查询
	resp, err = client.Send(&protocol.Message{
		ID:  "query-2",
		Cmd: protocol.CmdQuery,
		Filters: &protocol.LogFilter{
			Level: "error",
			Limit: 10,
		},
	})
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	printResponse(resp)

	fmt.Println("=== Test Completed ===")
}

// 压力测试
func stressLogTest(addr string, connections, requests int) {
	fmt.Printf("=== Log Stress Test: %d connections, %d requests each ===\n", connections, requests)

	var wg sync.WaitGroup
	var successCount int64
	var failedCount int64
	var totalLatency int64

	start := time.Now()

	// 创建连接池
	clients := make([]*Client, connections)
	for i := 0; i < connections; i++ {
		client, err := NewClient(addr)
		if err != nil {
			log.Printf("Failed to create client %d: %v", i, err)
			continue
		}
		clients[i] = client
	}

	// 发送日志
	for i, client := range clients {
		if client == nil {
			continue
		}
		wg.Add(1)
		go func(idx int, c *Client) {
			defer wg.Done()

			for j := 0; j < requests; j++ {
				// 随机生成日志级别
				levels := []string{"debug", "info", "warn", "error"}
				level := levels[j%4]

				reqStart := time.Now()
				resp, err := c.Send(&protocol.Message{
					ID:  fmt.Sprintf("conn%d-req%d", idx, j),
					Cmd: protocol.CmdLog,
					Data: map[string]interface{}{
						"timestamp": time.Now().UnixMilli(),
						"level":     level,
						"source":    fmt.Sprintf("client-%d", idx),
						"message":   fmt.Sprintf("Test log message %d", idx*requests+j),
						"tags":      []string{"stress-test"},
					},
				})
				latency := time.Since(reqStart).Milliseconds()

				if err != nil || resp.Status != "ok" {
					atomic.AddInt64(&failedCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
					atomic.AddInt64(&totalLatency, latency)
				}
			}
		}(i, client)
	}

	wg.Wait()
	duration := time.Since(start)

	// 关闭连接
	for _, c := range clients {
		if c != nil {
			c.Close()
		}
	}

	// 统计
	total := successCount + failedCount
	qps := float64(total) / duration.Seconds()
	avgLatency := float64(0)
	if successCount > 0 {
		avgLatency = float64(totalLatency) / float64(successCount)
	}

	fmt.Printf("\n=== Results ===\n")
	fmt.Printf("Duration:     %v\n", duration)
	fmt.Printf("Total:        %d\n", total)
	fmt.Printf("Success:      %d\n", successCount)
	fmt.Printf("Failed:       %d\n", failedCount)
	fmt.Printf("QPS:          %.2f\n", qps)
	fmt.Printf("Avg Latency:  %.2f ms\n", avgLatency)
}

// 心跳测试
func pingTest(addr string) {
	client, err := NewClient(addr)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fmt.Println("=== Ping Test ===")

	resp, err := client.Send(&protocol.Message{
		ID:  "ping-1",
		Cmd: protocol.CmdPing,
	})
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	printResponse(resp)
	fmt.Println("=== Test Completed ===")
}

func printResponse(resp *protocol.Response) {
	data, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("Response: %s\n\n", data)
}

func main() {
	var (
		addr      = flag.String("addr", "localhost:8080", "Server address")
		testType  = flag.String("type", "single", "Test type: single, batch, query, stress, ping")
		connCount = flag.Int("c", 10, "Connections for stress test")
		reqCount  = flag.Int("n", 100, "Requests per connection for stress test")
	)
	flag.Parse()

	switch *testType {
	case "single":
		singleLogTest(*addr)
	case "batch":
		batchLogTest(*addr)
	case "query":
		queryLogTest(*addr)
	case "stress":
		stressLogTest(*addr, *connCount, *reqCount)
	case "ping":
		pingTest(*addr)
	default:
		fmt.Println("Unknown test type. Use: single, batch, query, stress, ping")
		fmt.Println("\nExamples:")
		fmt.Println("  go run tcp_client.go -type=single")
		fmt.Println("  go run tcp_client.go -type=batch")
		fmt.Println("  go run tcp_client.go -type=query")
		fmt.Println("  go run tcp_client.go -type=stress -c 100 -n 1000")
		fmt.Println("  go run tcp_client.go -type=ping")
	}
}
