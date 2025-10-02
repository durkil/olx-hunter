package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"olx-hunter/internal/scraper"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisCache() *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		Password: "",
		DB: 0,
	})

	return &RedisCache{
		client: client,
		ctx: context.Background(),
	}
}

func (r *RedisCache) Ping() error {
	return r.client.Ping(r.ctx).Err()
}

func (r *RedisCache) CacheSearchResults(query string, results []scraper.Listing) error {
	key := fmt.Sprintf("scraping:%s", query)
	data, err := json.Marshal(results)
	if err != nil {
		return err
	}

	return r.client.Set(r.ctx, key, data, 10*time.Minute).Err()
}

func (r *RedisCache) GetCachedResults(query string) ([]scraper.Listing, bool) {
	key := fmt.Sprintf("scraping:%s", query)
	data, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return nil, false
	}
	if err != nil {
		return nil, false
	}

	var results []scraper.Listing
	err = json.Unmarshal([]byte(data), &results)
	if err != nil {
		return nil, false
	}

	return results, true
}

func (r *RedisCache) CanScrapeQuery(query string) bool {
	key := fmt.Sprintf("rate_limit:%s", query)
	count := r.client.Incr(r.ctx, key).Val()
	if count == 1 {
		r.client.Expire(r.ctx, key, 5*time.Minute)
	}
	return count == 1
}