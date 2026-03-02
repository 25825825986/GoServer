// Package pool Goroutine工作池实现
package pool

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"goserver/internal/protocol"
)

// Worker 工作协程
type Worker struct {
	id       int
	pool     *WorkerPool
	taskChan chan *protocol.Task
	quit     chan struct{}
}

// NewWorker 创建工作协程
func NewWorker(id int, pool *WorkerPool) *Worker {
	return &Worker{
		id:       id,
		pool:     pool,
		taskChan: make(chan *protocol.Task),
		quit:     make(chan struct{}),
	}
}

// Start 启动工作协程
func (w *Worker) Start() {
	go w.run()
}

// run 工作循环
func (w *Worker) run() {
	for {
		// 注册到工作池
		w.pool.workerPool <- w

		select {
		case task := <-w.taskChan:
			// 处理任务
			w.processTask(task)
		case <-w.quit:
			return
		case <-w.pool.ctx.Done():
			return
		}
	}
}

// processTask 处理任务
func (w *Worker) processTask(task *protocol.Task) {
	start := time.Now()

	// 调用处理器
	resp := w.pool.handler.Handle(task.ConnID, task.Conn, task.Message)
	
	// 计算延迟
	latency := time.Since(start).Milliseconds()
	resp.Latency = latency

	// 发送响应
	if err := task.Conn.Send(resp); err != nil {
		log.Printf("[Worker-%d] Send response error: %v", w.id, err)
	}

	// 更新统计
	atomic.AddUint64(&w.pool.stats.TasksProcessed, 1)
	atomic.AddInt64(&w.pool.stats.TotalLatency, latency)

	// 计算平均延迟
	processed := atomic.LoadUint64(&w.pool.stats.TasksProcessed)
	if processed > 0 {
		totalLatency := atomic.LoadInt64(&w.pool.stats.TotalLatency)
		avgLatency := float64(totalLatency) / float64(processed)
		atomic.StoreInt64(&w.pool.stats.AvgLatency, int64(avgLatency))
	}
}

// Stop 停止工作协程
func (w *Worker) Stop() {
	close(w.quit)
}

// WorkerPool 工作池
type WorkerPool struct {
	workers     []*Worker
	workerPool  chan *Worker
	taskQueue   chan *protocol.Task
	maxWorkers  int
	queueSize   int
	handler     protocol.Handler
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	
	// 统计
	stats       *PoolStats
}

// PoolStats 工作池统计
type PoolStats struct {
	TasksSubmitted uint64
	TasksProcessed uint64
	TasksFailed    uint64
	TotalLatency   int64
	AvgLatency     int64
	QueueLength    int32
}

// NewWorkerPool 创建工作池
func NewWorkerPool(maxWorkers, queueSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &WorkerPool{
		workers:    make([]*Worker, 0, maxWorkers),
		workerPool: make(chan *Worker, maxWorkers),
		taskQueue:  make(chan *protocol.Task, queueSize),
		maxWorkers: maxWorkers,
		queueSize:  queueSize,
		ctx:        ctx,
		cancel:     cancel,
		stats:      &PoolStats{},
	}
}

// SetHandler 设置处理器
func (p *WorkerPool) SetHandler(handler protocol.Handler) {
	p.handler = handler
}

// Start 启动工作池
func (p *WorkerPool) Start() {
	// 创建工作协程
	for i := 0; i < p.maxWorkers; i++ {
		worker := NewWorker(i, p)
		p.workers = append(p.workers, worker)
		worker.Start()
	}

	// 启动任务分发器
	p.wg.Add(1)
	go p.dispatcher()

	log.Printf("[WorkerPool] Started with %d workers, queue size %d", p.maxWorkers, p.queueSize)
}

// dispatcher 任务分发器
func (p *WorkerPool) dispatcher() {
	defer p.wg.Done()

	for {
		select {
		case task := <-p.taskQueue:
			// 等待可用工作协程
			select {
			case worker := <-p.workerPool:
				worker.taskChan <- task
			case <-p.ctx.Done():
				return
			}
		case <-p.ctx.Done():
			return
		}
	}
}

// SubmitTask 提交任务
func (p *WorkerPool) SubmitTask(task *protocol.Task) bool {
	select {
	case p.taskQueue <- task:
		atomic.AddUint64(&p.stats.TasksSubmitted, 1)
		return true
	default:
		// 队列满，任务被拒绝
		log.Printf("[WorkerPool] Task queue full, rejected")
		atomic.AddUint64(&p.stats.TasksFailed, 1)
		return false
	}
}

// Stop 停止工作池
func (p *WorkerPool) Stop() {
	p.cancel()

	// 等待分发器退出
	p.wg.Wait()

	// 停止所有工作协程
	for _, worker := range p.workers {
		worker.Stop()
	}

	log.Printf("[WorkerPool] Stopped")
}

// GetStats 获取统计
func (p *WorkerPool) GetStats() PoolStats {
	return PoolStats{
		TasksSubmitted: atomic.LoadUint64(&p.stats.TasksSubmitted),
		TasksProcessed: atomic.LoadUint64(&p.stats.TasksProcessed),
		TasksFailed:    atomic.LoadUint64(&p.stats.TasksFailed),
		TotalLatency:   atomic.LoadInt64(&p.stats.TotalLatency),
		AvgLatency:     atomic.LoadInt64(&p.stats.AvgLatency),
		QueueLength:    int32(len(p.taskQueue)),
	}
}
