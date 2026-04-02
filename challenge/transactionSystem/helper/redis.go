package helper

import (
	"belajar-go/challenge/transactionSystem/config"
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

func (m *RedisKeyManager) Generate(identifier string) string {
	return fmt.Sprintf("%s:%s:%s", m.service, m.domain, sanitize(identifier))
}

func sanitize(input string) string {
	res := strings.ToLower(input)
	res = strings.ReplaceAll(res, " ", "_")
	res = strings.ReplaceAll(res, "-", "_")
	return res
}

func SaveToCacheCompressed(ctx context.Context, rdb *redis.Client, key string, data any) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		PrintLog("cache", "Helper", "Gagal marshal JSON: "+err.Error())
		return
	}

	compressed, err := CompressData(jsonData)
	if err != nil {
		PrintLog("cache", "Helper", "Gagal kompresi: "+err.Error())
		return
	}

	err = rdb.Set(ctx, key, compressed, config.TimeCache).Err()
	if err != nil {
		PrintLog("redis", "Helper", "Peringatan: Gagal menyimpan cache: "+err.Error())
	}
}
