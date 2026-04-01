package helper

import (
	"fmt"
	"strings"
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
