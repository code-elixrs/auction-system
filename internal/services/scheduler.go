package services

import (
	"auction-system/internal/domain"
	"auction-system/pkg/logger"
	"auction-system/pkg/utils"
	"context"
	"time"

	"github.com/robfig/cron/v3"
)

type CronAuctionScheduler struct {
	cron       *cron.Cron
	repo       domain.SchedulerRepository
	auctionMgr *AuctionManager
	log        logger.Logger
}

func NewCronAuctionScheduler(repo domain.SchedulerRepository, auctionMgr *AuctionManager,
	log logger.Logger) *CronAuctionScheduler {
	return &CronAuctionScheduler{
		cron:       cron.New(cron.WithSeconds()),
		repo:       repo,
		auctionMgr: auctionMgr,
		log:        log,
	}
}

func (s *CronAuctionScheduler) Start(ctx context.Context) error {
	s.log.Info("Starting auction scheduler")

	// Add job to check for pending tasks every minute
	_, err := s.cron.AddFunc("@every 1m", func() {
		s.processPendingJobs(ctx)
	})

	if err != nil {
		return err
	}

	s.cron.Start()
	return nil
}

func (s *CronAuctionScheduler) Stop() error {
	s.log.Info("Stopping auction scheduler")
	s.cron.Stop()
	return nil
}

func (s *CronAuctionScheduler) ScheduleAuctionStart(ctx context.Context, auctionID string, startTime time.Time) error {
	job := &domain.ScheduledJob{
		ID:        utils.GenerateID("job"),
		AuctionID: auctionID,
		JobType:   domain.JobStartAuction,
		RunAt:     startTime,
		Status:    domain.JobPending,
		CreatedAt: time.Now(),
	}

	return s.repo.CreateJob(ctx, job)
}

func (s *CronAuctionScheduler) ScheduleAuctionEnd(ctx context.Context, auctionID string, endTime time.Time) error {
	job := &domain.ScheduledJob{
		ID:        utils.GenerateID("job"),
		AuctionID: auctionID,
		JobType:   domain.JobEndAuction,
		RunAt:     endTime,
		Status:    domain.JobPending,
		CreatedAt: time.Now(),
	}

	return s.repo.CreateJob(ctx, job)
}

func (s *CronAuctionScheduler) RescheduleAuctionEnd(ctx context.Context, auctionID string, newEndTime time.Time) error {
	// Cancel existing end jobs
	if err := s.repo.CancelJobsForAuction(ctx, auctionID); err != nil {
		return err
	}

	// Create new end job
	return s.ScheduleAuctionEnd(ctx, auctionID, newEndTime)
}

func (s *CronAuctionScheduler) CancelSchedule(ctx context.Context, auctionID string) error {
	return s.repo.CancelJobsForAuction(ctx, auctionID)
}

func (s *CronAuctionScheduler) processPendingJobs(ctx context.Context) {
	jobs, err := s.repo.GetPendingJobs(ctx, time.Now())
	if err != nil {
		s.log.Error("Failed to get pending jobs", "error", err)
		return
	}

	for _, job := range jobs {
		s.log.Info("Processing job", "job_id", job.ID, "type", job.JobType, "auction_id", job.AuctionID)

		var err error
		switch job.JobType {
		case domain.JobStartAuction:
			err = s.auctionMgr.StartAuction(ctx, job.AuctionID)
		case domain.JobEndAuction:
			err = s.auctionMgr.EndAuction(ctx, job.AuctionID)
		}

		status := domain.JobExecuted
		if err != nil {
			s.log.Error("Failed to execute job", "job_id", job.ID, "error", err)
			// Don't mark as executed on error, will retry
			continue
		}

		s.repo.UpdateJobStatus(ctx, job.ID, status)
	}
}
