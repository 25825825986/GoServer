package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config 系统配置结构体
type Config struct {
	Server ServerConfig
	Redis  RedisConfig
	App    AppConfig
}

// ServerConfig HTTP服务器配置
type ServerConfig struct {
	Port         string
	ReadTimeout  int
	WriteTimeout int
	MaxWorkers   int
}

// RedisConfig Redis连接配置
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	MaxRetry int
	PoolSize int
}

// AppConfig 应用配置
type AppConfig struct {
	Name        string
	Version     string
	Environment string
	BatchSize   int
	Workers     int
	QueueSize   int
}

// LoadConfig 从环境变量加载配置
func LoadConfig() (*Config, error) {
	// 加载.env文件（如果存在）
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  getEnvInt("SERVER_READ_TIMEOUT", 10),
			WriteTimeout: getEnvInt("SERVER_WRITE_TIMEOUT", 10),
			MaxWorkers:   getEnvInt("SERVER_MAX_WORKERS", 100),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
			MaxRetry: getEnvInt("REDIS_MAX_RETRY", 3),
			PoolSize: getEnvInt("REDIS_POOL_SIZE", 10),
		},
		App: AppConfig{
			Name:        getEnv("APP_NAME", "DataProcessor"),
			Version:     getEnv("APP_VERSION", "1.0.0"),
			Environment: getEnv("ENVIRONMENT", "development"),
			BatchSize:   getEnvInt("BATCH_SIZE", 100),
			Workers:     getEnvInt("WORKERS", 10),
			QueueSize:   getEnvInt("QUEUE_SIZE", 1000),
		},
	}

	return cfg, nil
}

// getEnv 获取环境变量，提供默认值
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvInt 获取整型环境变量，提供默认值
func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		var intVal int
		if n, _ := fmt.Sscanf(value, "%d", &intVal); n == 1 {
			return intVal
		}
	}
	return defaultValue
}
