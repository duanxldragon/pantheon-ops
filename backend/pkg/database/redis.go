package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client
var ctx = context.Background()

// InitRedis 初始化 Redis 连接
func InitRedis(addr string, password string, db int) {
	RDB = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // 如果没有密码则留空
		DB:       db,       // 默认使用 DB 0
	})

	// 测试连接
	_, err := RDB.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("failed to connect redis: %v", err)
	}

	fmt.Println("Redis connection successful")
}

// SetEx 设置带过期时间的缓存 (对底座后续业务很有用)
func SetEx(key string, value interface{}, expiration time.Duration) error {
	return RDB.Set(ctx, key, value, expiration).Err()
}

// Get 获取缓存
func Get(key string) (string, error) {
	return RDB.Get(ctx, key).Result()
}
