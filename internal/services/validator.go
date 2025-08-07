package services

import (
	"auction-system/internal/domain"
)

type BidValidatorImpl struct {
	biddingRuleDao domain.BiddingRuleDao
}

func NewBidValidator(dao domain.BiddingRuleDao) *BidValidatorImpl {
	return &BidValidatorImpl{
		biddingRuleDao: dao,
	}
}

func (v *BidValidatorImpl) ValidateIncrement(currentAmount, newAmount float64) bool {
	requiredIncrement := v.biddingRuleDao.GetIncrementRule(currentAmount)
	return newAmount >= currentAmount+requiredIncrement
}
