package domain

// Validation interface
type BidValidator interface {
	ValidateIncrement(currentAmount, newAmount float64) bool
}
