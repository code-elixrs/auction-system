package mysql

import (
	"context"
	"database/sql"
	"time"

	"auction-system/internal/domain"
)

type MySQLBidRepository struct {
	db *sql.DB
}

func NewMySQLBidRepository(db *sql.DB) *MySQLBidRepository {
	return &MySQLBidRepository{db: db}
}

func (r *MySQLBidRepository) SaveBidEvent(ctx context.Context, event *domain.BidEvent) error {
	query := `
        INSERT INTO bid_events (auction_id, user_id, amount, event_type, timestamp, created_at)
        VALUES (?, ?, ?, ?, ?, ?)
    `
	_, err := r.db.ExecContext(ctx, query,
		event.AuctionID, event.UserID, event.Amount,
		string(event.Type), event.Timestamp, time.Now())
	return err
}

func (r *MySQLBidRepository) GetBidHistory(ctx context.Context, auctionID string) ([]*domain.BidEvent, error) {
	query := `
        SELECT auction_id, user_id, amount, event_type, timestamp
        FROM bid_events 
        WHERE auction_id = ? AND event_type = 'bid_accepted'
        ORDER BY timestamp ASC
    `

	rows, err := r.db.QueryContext(ctx, query, auctionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.BidEvent
	for rows.Next() {
		var event domain.BidEvent
		var eventType string

		err := rows.Scan(&event.AuctionID, &event.UserID, &event.Amount,
			&eventType, &event.Timestamp)
		if err != nil {
			return nil, err
		}

		event.Type = domain.BidEventType(eventType)
		events = append(events, &event)
	}

	return events, nil
}
