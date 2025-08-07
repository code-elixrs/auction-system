package mysql

import (
	"auction-system/internal/domain"
	"context"
	"database/sql"
	"time"
)

type MySQLSchedulerRepository struct {
	db *sql.DB
}

func NewMySQLSchedulerRepository(db *sql.DB) *MySQLSchedulerRepository {
	return &MySQLSchedulerRepository{db: db}
}

func (r *MySQLSchedulerRepository) CreateJob(ctx context.Context, job *domain.ScheduledJob) error {
	query := `
        INSERT INTO scheduled_jobs (id, auction_id, job_type, run_at, status, created_at)
        VALUES (?, ?, ?, ?, ?, ?)
    `
	_, err := r.db.ExecContext(ctx, query,
		job.ID, job.AuctionID, string(job.JobType),
		job.RunAt, string(job.Status), job.CreatedAt)
	return err
}

func (r *MySQLSchedulerRepository) GetPendingJobs(ctx context.Context, before time.Time) ([]*domain.ScheduledJob, error) {
	query := `
        SELECT id, auction_id, job_type, run_at, status, created_at
        FROM scheduled_jobs 
        WHERE status = 'pending' AND run_at <= ?
        ORDER BY run_at ASC
    `

	rows, err := r.db.QueryContext(ctx, query, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*domain.ScheduledJob
	for rows.Next() {
		var job domain.ScheduledJob
		var jobType, status string

		err := rows.Scan(&job.ID, &job.AuctionID, &jobType,
			&job.RunAt, &status, &job.CreatedAt)
		if err != nil {
			return nil, err
		}

		job.JobType = domain.JobType(jobType)
		job.Status = domain.JobStatus(status)
		jobs = append(jobs, &job)
	}

	return jobs, nil
}

func (r *MySQLSchedulerRepository) UpdateJobStatus(ctx context.Context, jobID string, status domain.JobStatus) error {
	query := `UPDATE scheduled_jobs SET status = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, string(status), jobID)
	return err
}

func (r *MySQLSchedulerRepository) CancelJobsForAuction(ctx context.Context, auctionID string) error {
	query := `UPDATE scheduled_jobs SET status = 'cancelled' WHERE auction_id = ? AND status = 'pending'`
	_, err := r.db.ExecContext(ctx, query, auctionID)
	return err
}
