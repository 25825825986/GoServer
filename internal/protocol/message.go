// Package protocol 定义日志处理系统的TCP通信协议
package protocol

import (
	"encoding/json"
	"fmt"
)

// MessageType 消息类型
type MessageType string

const (
	// CmdLog 提交单条日志
	CmdLog MessageType = "log"
	// CmdBatch 批量提交日志
	CmdBatch MessageType = "batch"
	// CmdQuery 查询日志（用于交互式查询）
	CmdQuery MessageType = "query"
	// CmdPing 心跳检测
	CmdPing MessageType = "ping"
)

// LogLevel 日志级别
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
	LevelFatal LogLevel = "fatal"
)

// LogEntry 日志条目结构
type LogEntry struct {
	Timestamp int64       `json:"timestamp"`         // 时间戳（毫秒）
	Level     LogLevel    `json:"level"`             // 日志级别
	Source    string      `json:"source"`            // 来源（如：服务名/模块名）
	Message   string      `json:"message"`           // 日志内容
	Tags      []string    `json:"tags,omitempty"`    // 标签
	Metadata  interface{} `json:"metadata,omitempty"` // 额外元数据
}

// Message 请求消息结构
type Message struct {
	ID      string      `json:"id"`                 // 请求唯一ID
	Cmd     MessageType `json:"cmd"`                // 命令类型
	Data    interface{} `json:"data"`               // 数据负载（对于log类型，为LogEntry）
	Filters *LogFilter  `json:"filters,omitempty"`  // 查询过滤器（用于query命令）
}

// LogFilter 日志查询过滤器
type LogFilter struct {
	Level     string   `json:"level,omitempty"`     // 日志级别过滤
	Source    string   `json:"source,omitempty"`    // 来源过滤
	StartTime int64    `json:"start_time,omitempty"` // 开始时间戳
	EndTime   int64    `json:"end_time,omitempty"`   // 结束时间戳
	Keywords  []string `json:"keywords,omitempty"`   // 关键词过滤
	Limit     int      `json:"limit,omitempty"`      // 返回数量限制
}

// Response 响应消息结构
type Response struct {
	ID      string      `json:"id"`                // 对应请求ID
	Status  string      `json:"status"`            // ok 或 error
	Data    interface{} `json:"data,omitempty"`    // 响应数据
	Error   string      `json:"error,omitempty"`   // 错误信息（如果有）
	Latency int64       `json:"latency"`           // 处理延迟（毫秒）
	Count   int         `json:"count,omitempty"`   // 处理数量
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
