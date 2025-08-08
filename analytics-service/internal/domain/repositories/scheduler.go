package repositories

import (
	"auction-system/internal/domain"
	"context"
	"time"
)

type SchedulerRepository interface {
	CreateJob(ctx context.Context, job *domain.ScheduledJob) error
	GetPendingJobs(ctx context.Context, before time.Time) ([]*domain.ScheduledJob, error)
	UpdateJobStatus(ctx context.Context, jobID string, status domain.JobStatus) error
	CancelJobsForAuction(ctx context.Context, auctionID string) error
}
