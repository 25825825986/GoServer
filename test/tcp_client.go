// TCP客户端测试工具
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

// 单条测试
func singleTest(addr string) {
	client, err := NewClient(addr)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fmt.Println("=== Single Data Test ===")

	// 发送单条数据
	resp, err := client.Send(&protocol.Message{
		ID:  "1",
		Cmd: protocol.CmdProcess,
		Data: map[string]interface{}{
			"id":   "001",
			"name": "order_create",
			"amount": 299.99,
		},
	})
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	printResponse(resp)

	// 获取指标
	resp, err = client.Send(&protocol.Message{
		ID:  "2",
		Cmd: protocol.CmdGetMetrics,
	})
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	printResponse(resp)
}

// 批量测试
func batchTest(addr string) {
	client, err := NewClient(addr)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fmt.Println("=== Batch Data Test ===")

	data := []interface{}{
		map[string]interface{}{"id": "1", "name": "item1"},
		map[string]interface{}{"id": "2", "name": "item2"},
		map[string]interface{}{"id": "3", "name": "item3"},
		map[string]interface{}{"id": "4", "name": "item4"},
		map[string]interface{}{"id": "5", "name": "item5"},
	}

	resp, err := client.Send(&protocol.Message{
		ID:   "3",
		Cmd:  protocol.CmdBatch,
		Data: data,
	})
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	printResponse(resp)
}

// 发布订阅测试
func pubSubTest(addr string) {
	// 订阅者
	subscriber, err := NewClient(addr)
	if err != nil {
		log.Fatal(err)
	}
	defer subscriber.Close()

	// 订阅频道
	resp, err := subscriber.Send(&protocol.Message{
		ID:      "sub1",
		Cmd:     protocol.CmdSubscribe,
		Channel: "test_channel",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("=== Pub/Sub Test ===")
	printResponse(resp)

	// 启动goroutine接收推送
	go func() {
		for {
			line, err := subscriber.reader.ReadBytes('\n')
			if err != nil {
				return
			}
			var msg protocol.Response
			if err := json.Unmarshal(line, &msg); err != nil {
				continue
			}
			fmt.Printf("[Received Push] %+v\n", msg)
		}
	}()

	// 发布者
	publisher, err := NewClient(addr)
	if err != nil {
		log.Fatal(err)
	}
	defer publisher.Close()

	// 发布消息
	for i := 1; i <= 3; i++ {
		resp, err := publisher.Send(&protocol.Message{
			ID:      fmt.Sprintf("pub%d", i),
			Cmd:     protocol.CmdPublish,
			Channel: "test_channel",
			Data: map[string]interface{}{
				"message": fmt.Sprintf("Hello %d", i),
				"time":    time.Now().Format("15:04:05"),
			},
		})
		if err != nil {
			log.Printf("Publish error: %v", err)
			continue
		}
		printResponse(resp)
		time.Sleep(500 * time.Millisecond)
	}

	time.Sleep(1 * time.Second)
}

// 压力测试
func stressTest(addr string, connections, requests int) {
	fmt.Printf("=== Stress Test: %d connections, %d requests each ===\n", connections, requests)

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

	// 发送请求
	for i, client := range clients {
		if client == nil {
			continue
		}
		wg.Add(1)
		go func(idx int, c *Client) {
			defer wg.Done()

			for j := 0; j < requests; j++ {
				reqStart := time.Now()
				resp, err := c.Send(&protocol.Message{
					ID:  fmt.Sprintf("conn%d-req%d", idx, j),
					Cmd: protocol.CmdProcess,
					Data: map[string]interface{}{
						"index": idx*requests + j,
						"time":  time.Now().UnixNano(),
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
	avgLatency := float64(totalLatency) / float64(successCount)

	fmt.Printf("\n=== Results ===\n")
	fmt.Printf("Duration:     %v\n", duration)
	fmt.Printf("Total:        %d\n", total)
	fmt.Printf("Success:      %d\n", successCount)
	fmt.Printf("Failed:       %d\n", failedCount)
	fmt.Printf("QPS:          %.2f\n", qps)
	fmt.Printf("Avg Latency:  %.2f ms\n", avgLatency)
}

func printResponse(resp *protocol.Response) {
	data, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("Response: %s\n\n", data)
}

func main() {
	var (
		addr       = flag.String("addr", "localhost:8080", "Server address")
		testType   = flag.String("type", "single", "Test type: single, batch, pubsub, stress")
		connCount  = flag.Int("c", 100, "Connections for stress test")
		reqCount   = flag.Int("n", 100, "Requests per connection for stress test")
	)
	flag.Parse()

	switch *testType {
	case "single":
		singleTest(*addr)
	case "batch":
		batchTest(*addr)
	case "pubsub":
		pubSubTest(*addr)
	case "stress":
		stressTest(*addr, *connCount, *reqCount)
	default:
		fmt.Println("Unknown test type. Use: single, batch, pubsub, stress")
	}
}
