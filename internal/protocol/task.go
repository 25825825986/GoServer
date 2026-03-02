package protocol

import "time"

// ConnInterface 连接接口（用于避免循环依赖）
type ConnInterface interface {
	Send(resp *Response) error
	GetID() string
}

// Handler 消息处理器接口
type Handler interface {
	Handle(connID string, conn ConnInterface, msg *Message) *Response
}

// Task 任务结构
type Task struct {
	ConnID     string
	Message    *Message
	Conn       ConnInterface
	SubmitTime time.Time
}
