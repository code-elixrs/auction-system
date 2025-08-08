package domain

import "context"

// Leader election interface
type LeaderElection interface {
	BecomeLeader(ctx context.Context, instanceID string) (bool, error)
	IsLeader(ctx context.Context, instanceID string) (bool, error)
	ReleaseLeadership(ctx context.Context, instanceID string) error
}
