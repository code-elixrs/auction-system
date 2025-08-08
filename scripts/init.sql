-- Set session settings for better compatibility
SET SESSION sql_mode = 'STRICT_TRANS_TABLES,NO_ZERO_DATE,NO_ZERO_IN_DATE,ERROR_FOR_DIVISION_BY_ZERO';
SET SESSION time_zone = '+00:00';

-- Create database if it doesn't exist
CREATE DATABASE IF NOT EXISTS auction_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Use the database
USE auction_db;

-- Drop existing tables if they exist (for clean restart)
DROP TABLE IF EXISTS bid_events;
DROP TABLE IF EXISTS scheduled_jobs;
DROP TABLE IF EXISTS auctions;

-- Create auctions table
CREATE TABLE auctions (
                          id VARCHAR(255) PRIMARY KEY,
                          start_time TIMESTAMP NOT NULL,
                          end_time TIMESTAMP NOT NULL,
                          start_bid   FLOAT NOT NULL,
                          status INT NOT NULL DEFAULT 0 COMMENT '0=pending, 1=active, 2=ended, 3=cancelled',
                          created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                          updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                          INDEX idx_status (status),
                          INDEX idx_start_time (start_time),
                          INDEX idx_end_time (end_time),
                          INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create bid events table for analytics
CREATE TABLE bid_events (
                            id BIGINT AUTO_INCREMENT PRIMARY KEY,
                            auction_id VARCHAR(255) NOT NULL,
                            user_id VARCHAR(255) NOT NULL,
                            amount DECIMAL(15,2) NOT NULL,
                            event_type VARCHAR(50) NOT NULL,
                            timestamp TIMESTAMP NOT NULL,
                            created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                            INDEX idx_auction_id (auction_id),
                            INDEX idx_user_id (user_id),
                            INDEX idx_timestamp (timestamp),
                            INDEX idx_event_type (event_type),
                            INDEX idx_created_at (created_at),
                            FOREIGN KEY (auction_id) REFERENCES auctions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create scheduled jobs table
CREATE TABLE scheduled_jobs (
                                id VARCHAR(255) PRIMARY KEY,
                                auction_id VARCHAR(255) NOT NULL,
                                job_type VARCHAR(50) NOT NULL COMMENT 'start_auction, end_auction',
                                run_at TIMESTAMP NOT NULL,
                                status VARCHAR(50) NOT NULL DEFAULT 'pending' COMMENT 'pending, executed, cancelled',
                                created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                INDEX idx_auction_id (auction_id),
                                INDEX idx_run_at (run_at),
                                INDEX idx_status (status),
                                INDEX idx_job_type (job_type),
                                INDEX idx_created_at (created_at),
                                FOREIGN KEY (auction_id) REFERENCES auctions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Insert sample data for testing (optional)
INSERT INTO auctions (id, start_time, end_time, status, created_at, updated_at) VALUES
    ('auction_sample_001', NOW() + INTERVAL 5 MINUTE, NOW() + INTERVAL 65 MINUTE, 0, NOW(), NOW());

-- Create user for application with proper permissions
-- Note: This user is already created via environment variables, but we ensure permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON auction_db.* TO 'auction_user'@'%';
FLUSH PRIVILEGES;

-- Verify tables were created
SHOW TABLES;

-- Show table structures for verification
DESCRIBE auctions;
DESCRIBE bid_events;
DESCRIBE scheduled_jobs;

-- Print success message
SELECT 'Database initialization completed successfully!' as message;