package domain

import "context"

type BiddingRule interface {
	GetMinimumBid(currentAmount float64) float64
	GetIncrementRule(amount float64) float64
	LoadRules(ctx context.Context) error
}
