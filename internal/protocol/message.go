// Package protocol 定义TCP通信协议
package protocol

import (
	"encoding/json"
	"fmt"
)

// MessageType 消息类型
type MessageType string

const (
	// CmdProcess 处理单条数据
	CmdProcess MessageType = "process"
	// CmdBatch 批量处理
	CmdBatch MessageType = "batch"
	// CmdPublish 发布消息到队列
	CmdPublish MessageType = "publish"
	// CmdSubscribe 订阅队列
	CmdSubscribe MessageType = "subscribe"
	// CmdPing 心跳检测
	CmdPing MessageType = "ping"
	// CmdPong 心跳响应
	CmdPong MessageType = "pong"
	// CmdGetMetrics 获取指标
	CmdGetMetrics MessageType = "metrics"
	// CmdGetConfig 获取配置
	CmdGetConfig MessageType = "config"
)

// Message 请求消息结构
type Message struct {
	ID      string      `json:"id"`      // 请求唯一ID
	Cmd     MessageType `json:"cmd"`     // 命令类型
	Channel string      `json:"channel"` // 队列/频道名称（可选）
	Data    interface{} `json:"data"`    // 数据负载
}

// Response 响应消息结构
type Response struct {
	ID      string      `json:"id"`      // 对应请求ID
	Status  string      `json:"status"`  // ok 或 error
	Data    interface{} `json:"data"`    // 响应数据
	Error   string      `json:"error"`   // 错误信息（如果有）
	Latency int64       `json:"latency"` // 处理延迟（毫秒）
}

// Marshal 序列化消息
func (m *Message) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

// UnmarshalMessage 反序列化消息
func UnmarshalMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal message failed: %w", err)
	}
	return &msg, nil
}

// Marshal 序列化响应
func (r *Response) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// NewResponse 创建成功响应
func NewResponse(id string, data interface{}) *Response {
	return &Response{
		ID:     id,
		Status: "ok",
		Data:   data,
	}
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(id string, errMsg string) *Response {
	return &Response{
		ID:     id,
		Status: "error",
		Error:  errMsg,
	}
}

// Protocol 协议处理器接口
type Protocol interface {
	Decode(data []byte) (*Message, error)
	Encode(resp *Response) ([]byte, error)
}

// JSONProtocol JSON协议实现
type JSONProtocol struct{}

// Decode 解码消息
func (p *JSONProtocol) Decode(data []byte) (*Message, error) {
	return UnmarshalMessage(data)
}

// Encode 编码响应
func (p *JSONProtocol) Encode(resp *Response) ([]byte, error) {
	return resp.Marshal()
}

// NewJSONProtocol 创建JSON协议处理器
func NewJSONProtocol() *JSONProtocol {
	return &JSONProtocol{}
}
