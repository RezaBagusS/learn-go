package config

import (
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	log.Println(os.Getenv("REDIS_HOST"))
	log.Println(os.Getenv("REDIS_PORT"))
	log.Println(os.Getenv("REDIS_PASSWORD"))
	return rdb
}
