package redis

import (
	"auction-system/internal/domain"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisBidCache struct {
	client *redis.Client
}

func NewRedisBidCache(client *redis.Client) *RedisBidCache {
	return &RedisBidCache{client: client}
}

func (r *RedisBidCache) InitializeAuction(ctx context.Context, auctionID string, startingBid float64, incrementRule float64) error {
	key := fmt.Sprintf("auction:%s", auctionID)

	return r.client.HMSet(ctx, key,
		"current_bid", fmt.Sprintf("%.2f", startingBid),
		"winner_id", "",
		"increment_rule", fmt.Sprintf("%.2f", incrementRule),
		"last_updated", time.Now().Unix(),
	).Err()
}

func (r *RedisBidCache) AtomicBidUpdate(ctx context.Context, auctionID, userID string, amount float64) (bool, error) {
	luaScript := `
        local auction_key = "auction:" .. KEYS[1]
        local current_amount = redis.call('HGET', auction_key, 'current_bid')
        local increment_rule = redis.call('HGET', auction_key, 'increment_rule')
        
        if current_amount == false then
            return {0, "auction_not_found"}
        end
        
        local current = tonumber(current_amount)
        local new_amount = tonumber(ARGV[1])
        local required_increment = tonumber(increment_rule or "5")
        
        if new_amount >= (current + required_increment) then
            redis.call('HSET', auction_key, 
                'current_bid', ARGV[1], 
                'winner_id', ARGV[2], 
                'last_updated', ARGV[3])
            
            local event_data = KEYS[1] .. ":" .. "bid_accepted" .. ":" .. ARGV[2] .. ":" .. ARGV[1] .. ":" .. ARGV[3]
            redis.call('PUBLISH', 'auction_events', event_data)
            
            return {1, "success"}
        else
            local event_data = KEYS[1] .. ":" .. "bid_rejected" .. ":" .. ARGV[2] .. ":" .. ARGV[1] .. ":" .. ARGV[3]
            redis.call('PUBLISH', 'auction_events', event_data)
            
            return {0, "insufficient_increment"}
        end
    `

	result, err := r.client.Eval(ctx, luaScript, []string{auctionID},
		fmt.Sprintf("%.2f", amount),
		userID,
		strconv.FormatInt(time.Now().Unix(), 10)).Result()

	if err != nil {
		return false, err
	}

	resultSlice := result.([]interface{})
	return resultSlice[0].(int64) == 1, nil
}

func (r *RedisBidCache) GetCurrentBid(ctx context.Context, auctionID string) (*domain.LocalAuctionCache, error) {
	key := fmt.Sprintf("auction:%s", auctionID)

	result, err := r.client.HMGet(ctx, key, "current_bid", "winner_id", "increment_rule").Result()
	if err != nil {
		return nil, err
	}

	currentBid := 0.0
	winnerID := ""
	incrementRule := 5.0

	if result[0] != nil {
		currentBid, _ = strconv.ParseFloat(result[0].(string), 64)
	}
	if result[1] != nil {
		winnerID = result[1].(string)
	}
	if result[2] != nil {
		incrementRule, _ = strconv.ParseFloat(result[2].(string), 64)
	}

	return &domain.LocalAuctionCache{
		AuctionID:     auctionID,
		CurrentBid:    currentBid,
		WinnerID:      winnerID,
		IncrementRule: incrementRule,
		LastUpdated:   time.Now(),
	}, nil
}

func (r *RedisBidCache) SetAuctionIncrementRule(ctx context.Context, auctionID string, rule float64) error {
	key := fmt.Sprintf("auction:%s", auctionID)
	return r.client.HSet(ctx, key, "increment_rule", fmt.Sprintf("%.2f", rule)).Err()
}
