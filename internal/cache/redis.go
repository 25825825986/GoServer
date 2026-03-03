package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"goserver/internal/config"

	"github.com/redis/go-redis/v9"
)

// RedisClient Redis客户端的包装
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient 创建新的Redis客户端
func NewRedisClient(cfg *config.Config) *RedisClient {
	addr := fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port)
	
	client := redis.NewClient(&redis.Options{
		Addr:      addr,
		Password:  cfg.Redis.Password,
		DB:        cfg.Redis.DB,
		MaxRetries: cfg.Redis.MaxRetry,
		PoolSize:  cfg.Redis.PoolSize,
		DialTimeout: 5 * time.Second,
		ReadTimeout: 3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	return &RedisClient{client: client}
}

// Ping 测试Redis连接
func (rc *RedisClient) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return rc.client.Ping(ctx).Err()
}

// Set 设置键值对
func (rc *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// 自动将非字符串类型序列化为JSON
	var storeValue interface{}
	switch v := value.(type) {
	case string, []byte:
		storeValue = v
	default:
		jsonData, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
		storeValue = jsonData
	}
	return rc.client.Set(ctx, key, storeValue, expiration).Err()
}

// Get 获取值
func (rc *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return rc.client.Get(ctx, key).Result()
}

// Del 删除键
func (rc *RedisClient) Del(ctx context.Context, keys ...string) (int64, error) {
	return rc.client.Del(ctx, keys...).Result()
}

// Incr 递增
func (rc *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	return rc.client.Incr(ctx, key).Result()
}

// IncrBy 增加指定值
func (rc *RedisClient) IncrBy(ctx context.Context, key string, increment int64) (int64, error) {
	return rc.client.IncrBy(ctx, key, increment).Result()
}

// Lpush 左侧推入
func (rc *RedisClient) Lpush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	return rc.client.LPush(ctx, key, values...).Result()
}

// Rpop 右侧弹出
func (rc *RedisClient) Rpop(ctx context.Context, key string) (string, error) {
	return rc.client.RPop(ctx, key).Result()
}

// LRange 获取列表范围内的元素
func (rc *RedisClient) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return rc.client.LRange(ctx, key, start, stop).Result()
}

// LTrim 修剪列表
func (rc *RedisClient) LTrim(ctx context.Context, key string, start, stop int64) error {
	return rc.client.LTrim(ctx, key, start, stop).Err()
}

// LLen 获取列表长度
func (rc *RedisClient) LLen(ctx context.Context, key string) (int64, error) {
	return rc.client.LLen(ctx, key).Result()
}

// Sadd 集合添加
func (rc *RedisClient) Sadd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return rc.client.SAdd(ctx, key, members...).Result()
}

// Smembers 获取集合所有成员
func (rc *RedisClient) Smembers(ctx context.Context, key string) ([]string, error) {
	return rc.client.SMembers(ctx, key).Result()
}

// Hset 哈希表设置
func (rc *RedisClient) Hset(ctx context.Context, key string, field string, value interface{}) (int64, error) {
	return rc.client.HSet(ctx, key, field, value).Result()
}

// Hget 哈希表获取
func (rc *RedisClient) Hget(ctx context.Context, key string, field string) (string, error) {
	return rc.client.HGet(ctx, key, field).Result()
}

// Hgetall 哈希表获取所有
func (rc *RedisClient) Hgetall(ctx context.Context, key string) (map[string]string, error) {
	return rc.client.HGetAll(ctx, key).Result()
}

// Keys 获取所有键
func (rc *RedisClient) Keys(ctx context.Context, pattern string) ([]string, error) {
	return rc.client.Keys(ctx, pattern).Result()
}

// Exists 检查键是否存在
func (rc *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	return rc.client.Exists(ctx, keys...).Result()
}

// Expire 设置过期时间
func (rc *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return rc.client.Expire(ctx, key, expiration).Result()
}

// Close 关闭Redis连接
func (rc *RedisClient) Close() error {
	if rc.client != nil {
		return rc.client.Close()
	}
	return nil
}

// GetClient 获取底层Redis客户端
func (rc *RedisClient) GetClient() *redis.Client {
	return rc.client
}
