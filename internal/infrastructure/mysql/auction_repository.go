package mysql

import (
	"auction-system/internal/domain"
	"context"
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLAuctionRepository struct {
	db *sql.DB
}

func NewMySQLAuctionRepository(db *sql.DB) *MySQLAuctionRepository {
	return &MySQLAuctionRepository{db: db}
}

func (r *MySQLAuctionRepository) CreateAuction(ctx context.Context, auction *domain.Auction) error {
	query := `
        INSERT INTO auctions (id, start_time, end_time, status, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?)
    `
	_, err := r.db.ExecContext(ctx, query,
		auction.ID, auction.StartTime, auction.EndTime,
		int(auction.Status), auction.CreatedAt, auction.UpdatedAt)
	return err
}

func (r *MySQLAuctionRepository) GetAuction(ctx context.Context, auctionID string) (*domain.Auction, error) {
	query := `
        SELECT id, start_time, end_time, status, created_at, updated_at
        FROM auctions WHERE id = ?
    `

	var auction domain.Auction
	var status int

	err := r.db.QueryRowContext(ctx, query, auctionID).Scan(
		&auction.ID, &auction.StartTime, &auction.EndTime,
		&status, &auction.CreatedAt, &auction.UpdatedAt)

	if err != nil {
		return nil, err
	}

	auction.Status = domain.AuctionStatus(status)
	return &auction, nil
}

func (r *MySQLAuctionRepository) UpdateAuctionStatus(ctx context.Context, auctionID string, status domain.AuctionStatus) error {
	query := `UPDATE auctions SET status = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, int(status), time.Now(), auctionID)
	return err
}

func (r *MySQLAuctionRepository) GetActiveAuctions(ctx context.Context) ([]*domain.Auction, error) {
	query := `
        SELECT id, start_time, end_time, status, created_at, updated_at
        FROM auctions WHERE status = ?
    `

	rows, err := r.db.QueryContext(ctx, query, int(domain.AuctionActive))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var auctions []*domain.Auction
	for rows.Next() {
		var auction domain.Auction
		var status int

		err := rows.Scan(&auction.ID, &auction.StartTime, &auction.EndTime,
			&status, &auction.CreatedAt, &auction.UpdatedAt)
		if err != nil {
			return nil, err
		}

		auction.Status = domain.AuctionStatus(status)
		auctions = append(auctions, &auction)
	}

	return auctions, nil
}
