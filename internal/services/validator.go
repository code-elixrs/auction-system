package services

import (
	"auction-system/internal/domain"
	"context"
	"encoding/json"

	"github.com/go-redis/redis/v8"
)

type RedisBidValidator struct {
	client *redis.Client
	rules  *domain.BidValidationRules
}

func NewRedisBidValidator(client *redis.Client) *RedisBidValidator {
	return &RedisBidValidator{
		client: client,
	}
}

func (v *RedisBidValidator) LoadRules(ctx context.Context) error {
	data, err := v.client.Get(ctx, "bid_validation_rules").Result()
	if err != nil {
		if err == redis.Nil {
			// Set default rules
			v.rules = &domain.BidValidationRules{
				Rules: map[string]float64{
					"0-100":   5.0,
					"100-500": 10.0,
					"500+":    25.0,
				},
			}
			// Save to Redis
			return v.saveRules(ctx)
		}
		return err
	}

	var rules domain.BidValidationRules
	if err := json.Unmarshal([]byte(data), &rules); err != nil {
		return err
	}

	v.rules = &rules
	return nil
}

func (v *RedisBidValidator) saveRules(ctx context.Context) error {
	data, err := json.Marshal(v.rules)
	if err != nil {
		return err
	}

	return v.client.Set(ctx, "bid_validation_rules", string(data), 0).Err()
}

func (v *RedisBidValidator) ValidateIncrement(currentAmount, newAmount float64) bool {
	if v.rules == nil {
		return false
	}

	requiredIncrement := v.GetIncrementRule(currentAmount)
	return newAmount >= currentAmount+requiredIncrement
}

func (v *RedisBidValidator) GetMinimumBid(currentAmount float64) float64 {
	return currentAmount + v.GetIncrementRule(currentAmount)
}

func (v *RedisBidValidator) GetIncrementRule(amount float64) float64 {
	if v.rules == nil {
		return 5.0 // default
	}

	if amount < 100 {
		return v.rules.Rules["0-100"]
	} else if amount < 500 {
		return v.rules.Rules["100-500"]
	} else {
		return v.rules.Rules["500+"]
	}
}
