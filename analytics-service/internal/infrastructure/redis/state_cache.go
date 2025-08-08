package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"auction-system/internal/domain"

	"github.com/go-redis/redis/v8"
)

type StateCacheImpl struct {
	client *redis.Client
}

func NewStateCache(client *redis.Client) *StateCacheImpl {
	return &StateCacheImpl{client: client}
}

func (r *StateCacheImpl) SetAuctionStatus(ctx context.Context, auctionID string,
	status domain.AuctionStatus) error {
	key := fmt.Sprintf("auction:%s:status", auctionID)
	return r.client.Set(ctx, key, int(status), 0).Err()
}

func (r *StateCacheImpl) GetAuctionStatus(ctx context.Context,
	auctionID string) (domain.AuctionStatus, error) {
	key := fmt.Sprintf("auction:%s:status", auctionID)

	result, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
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
