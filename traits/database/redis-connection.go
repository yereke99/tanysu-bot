package database

import (
	"tanysu-bot/config"

	"github.com/redis/go-redis/v9"
)

func RedisConnection(cfg *config.Config) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,     // Адрес Redis
		Password: cfg.RedisPassword, // Пароль Redis (если требуется)
		DB:       cfg.RedisDB,       // Номер базы данных Redis
	})

	return client
}
