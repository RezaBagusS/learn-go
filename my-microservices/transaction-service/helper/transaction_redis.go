package helper

import (
	"my-microservices/transaction-service/config"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

type RedisKeyManager struct {
	service string
	domain  string
}

func NewRedisKeyManager(service, domain string) *RedisKeyManager {
	return &RedisKeyManager{
		service: sanitize(service),
		domain:  sanitize(domain),
	}
}

func (m *RedisKeyManager) Generate(identifier string, parts ...string) string {
	if len(parts) == 0 {
		return fmt.Sprintf("%s:%s:%s", m.service, m.domain, sanitize(identifier))
	}

	sanitizedParts := make([]string, len(parts))
	for i, p := range parts {
		sanitizedParts[i] = sanitize(p)
	}

	return fmt.Sprintf("%s:%s:%s:%s", m.service, m.domain, sanitize(identifier), strings.Join(sanitizedParts, ":"))
}

func sanitize(input string) string {
	res := strings.ToLower(input)
	res = strings.ReplaceAll(res, " ", "_")
	res = strings.ReplaceAll(res, "-", "_")
	return res
}

func SaveToCacheCompressed(ctx context.Context, rdb *redis.Client, key string, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	compressed, err := CompressData(jsonData)
	if err != nil {
		return err
	}

	return rdb.Set(ctx, key, compressed, config.TimeCache).Err()
}
