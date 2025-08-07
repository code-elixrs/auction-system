package redis

import (
	"auction-system/internal/domain"
	"context"
	"fmt"
	"strconv"

	"github.com/go-redis/redis/v8"
)

type RedisStateCache struct {
	client *redis.Client
}

func NewRedisStateCache(client *redis.Client) *RedisStateCache {
	return &RedisStateCache{client: client}
}

func (r *RedisStateCache) SetAuctionStatus(ctx context.Context, auctionID string, status domain.AuctionStatus) error {
	key := fmt.Sprintf("auction:%s:status", auctionID)
	return r.client.Set(ctx, key, int(status), 0).Err()
}

func (r *RedisStateCache) GetAuctionStatus(ctx context.Context, auctionID string) (domain.AuctionStatus, error) {
	key := fmt.Sprintf("auction:%s:status", auctionID)

	result, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return domain.AuctionPending, nil
		}
		return domain.AuctionPending, err
	}

	status, err := strconv.Atoi(result)
	if err != nil {
		return domain.AuctionPending, err
	}

	return domain.AuctionStatus(status), nil
}
