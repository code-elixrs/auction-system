package services

import (
	"context"
	"encoding/json"
	"errors"

	"auction-system/internal/domain"

	"github.com/go-redis/redis/v8"
)

type BiddingRuleDaoImpl struct {
	client *redis.Client
	rules  *domain.BidValidationRules
}

func NewBiddingRuleDao(client *redis.Client) *BiddingRuleDaoImpl {
	return &BiddingRuleDaoImpl{
		client: client,
	}
}

func (v *BiddingRuleDaoImpl) LoadRules(ctx context.Context) error {
	data, err := v.client.Get(ctx, "bid_validation_rules").Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// TODO: Ideally this should be loaded into redis from config or some db.
			// TODO: This should be configurable at an auction level and it should be a step during auction setup.
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

func (v *BiddingRuleDaoImpl) saveRules(ctx context.Context) error {
	data, err := json.Marshal(v.rules)
	if err != nil {
		return err
	}

	return v.client.Set(ctx, "bid_validation_rules", string(data), 0).Err()
}

func (v *BiddingRuleDaoImpl) GetMinimumBid(currentAmount float64) float64 {
	return currentAmount + v.GetIncrementRule(currentAmount)
}

func (v *BiddingRuleDaoImpl) GetIncrementRule(amount float64) float64 {
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
