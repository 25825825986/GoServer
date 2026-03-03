// Package models 数据模型
package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
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

// StringArray 字符串数组类型（用于数据库存储JSON）
type StringArray []string

// Value 实现 driver.Valuer 接口
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return "[]", nil
	}
	return json.Marshal(a)
}

// Scan 实现 sql.Scanner 接口
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = StringArray{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}

	return json.Unmarshal(bytes, a)
}

// JSONMap JSON对象类型
type JSONMap map[string]interface{}

// Value 实现 driver.Valuer 接口
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	return json.Marshal(m)
}

// Scan 实现 sql.Scanner 接口
func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = JSONMap{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}

	return json.Unmarshal(bytes, m)
}

// LogEntry 日志条目模型
type LogEntry struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Timestamp int64     `gorm:"index:idx_time;not null" json:"timestamp"`
	Level     LogLevel  `gorm:"type:varchar(20);index:idx_level" json:"level"`
	Source    string    `gorm:"type:varchar(255);index:idx_source" json:"source"`
	Message   string    `gorm:"type:text;not null" json:"message"`
	Tags      StringArray `gorm:"type:json" json:"tags"`
	Metadata  JSONMap   `gorm:"type:json" json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 设置表名
func (LogEntry) TableName() string {
	return "logs"
}

// ToMap 转换为map
func (l *LogEntry) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":        l.ID,
		"timestamp": l.Timestamp,
		"level":     l.Level,
		"source":    l.Source,
		"message":   l.Message,
		"tags":      l.Tags,
		"metadata":  l.Metadata,
	}
}

// LogStats 日志统计
type LogStats struct {
	Date       string         `json:"date"`
	Total      int64          `json:"total"`
	ByLevel    map[string]int64 `json:"by_level"`
}
