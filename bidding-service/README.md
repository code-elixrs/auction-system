# Distributed Auction System

A high-performance, low-latency distributed auction system built with Go, Redis, and MySQL.

## Architecture

- **Bidding Service**: Handles WebSocket connections and bid processing
- **Auction Service**: Manages auction lifecycle and scheduling
- **Analytics Service**: Processes and stores bid events for analytics

## Features

- ✅ Real-time bidding via WebSockets
- ✅ Atomic bid validation and updates using Redis Lua scripts
- ✅ Distributed leader election for auction management
- ✅ [TODO:]30-second auction extension on last-minute bids
- ✅ Event-driven architecture with Redis pub/sub
- ✅ Configurable bid increment rules
- ✅ Horizontal scaling support
- ✅ [TODO:]Circuit breaker pattern ready
- ✅ [IN_PROGRESS]Comprehensive logging and monitoring

## Quick Start

### Prerequisites

- Go 1.24+
- Docker & Docker Compose
- Make (optional)

### Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/code-elixrs/auction-system
   cd auction-system
   ```

2. **Start infrastructure services**
   ```bash
   make dev-setup
   ```

3. **Run services locally**
   ```bash
   # Terminal 1: Auction Service
   make run-auction-service
   
   # Terminal 2: Bidding Service  
   make run-bidding-service
   
   # Terminal 3: Analytics Service
   make run-analytics-service
   ```
4. **Run UI locally**
    ```bash
    python3 -m http.server 3000 -d web/
   ```
### Docker Deployment

1. **Start all services**
   ```bash
   make docker-up
   ```

2. **View logs**
   ```bash
   make docker-logs
   ```

3. **Stop services**
   ```bash
   make docker-down
   ```

## API Usage

### Create Auction
```bash
curl -X POST http://localhost:8081/api/v1/auctions \
  -H "Content-Type: application/json" \
  -d '{
    "start_time": "2024-12-01T10:00:00Z",
    "end_time": "2024-12-01T12:00:00Z",
    "starting_bid": 100.0
  }'
```

### Connect to Auction (WebSocket)
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/auction/auction_123?user_id=user_456');

// Place a bid
ws.send(JSON.stringify({
  type: 'place_bid',
  amount: '105.00'
}));

// Listen for updates
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Auction update:', data);
};
```

## Configuration

Configuration can be provided via `config.yaml` file or environment variables:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

redis:
  address: "localhost:6379"
  password: ""
  db: 0

mysql:
  dsn: "user:pass@tcp(localhost:3306)/auction_db?parseTime=true"
  max_open_conns: 25
  max_idle_conns: 10
  conn_max_lifetime: "5m"
```

## Architecture Details

https://docs.google.com/document/d/1OOUcQYz0rmgpxeSC_MJf02pMUwgrB6K18NdMR7xBMHs/edit?tab=t.0#heading=h.xo5irxaku2gu

### Bid Flow
1. User places bid via WebSocket
2. Quick local validation using cached data
3. Atomic Redis update via Lua script
4. Redis publishes event to all nodes
5. Event listeners update local caches
6. WebSocket notifications sent to all participants

### Leader Election
- Redis-based leader election with TTL
- Only leader can start/end auctions
- Automatic failover on leader failure

### Data Consistency
- Redis as single source of truth for active auctions
- Async MySQL persistence for durability
- Event-driven eventual consistency

## Scaling

- **Horizontal**: Add more auction/bidding service instances
- **WebSocket**: Each node handles independent connections
- **Database**: MySQL read replicas, Redis clustering
- **Load Balancing**: Use nginx/HAProxy for WebSocket sticky sessions

## Monitoring

- Health checks: `GET /health`
- Structured JSON logging
- TODO: Metrics ready (Prometheus integration possible)

## Development

```bash
# Format code
make fmt

```

## Service Details

### Bidding Service (Port 8080)
- **Purpose**: Handle WebSocket connections and bid processing
- **Endpoints**:
    - `GET /health` - Health check
    - `WS /ws/auction/{auctionID}?user_id={userID}` - WebSocket connection
- **Responsibilities**:
    - WebSocket connection management
    - Real-time bid processing
    - Local auction cache management
    - Event listening and broadcasting

### Auction Manager (Port 8081)
- **Purpose**: Manage auction lifecycle and scheduling
- **Endpoints**:
    - `POST /api/v1/auctions` - Create auction
    - `GET /api/v1/auctions/{id}` - Get auction details
    - `POST /api/v1/auctions/{id}/extend` - Extend auction
    - `GET /health` - Health check
- **Responsibilities**:
    - Auction creation and management
    - Distributed job scheduling
    - Leader election
    - Auction state transitions

### Analytics Service (Background)
- **Purpose**: Process and store bid events for analytics
- **Responsibilities**:
    - Subscribe to Redis pub/sub events
    - Store successful bid events to MySQL
    - Extensible for future analytics features


## Project Structure

```
.
├── cmd                         # Main applications
│   ├── analytics-service       # Event analytics service
│   ├── auction-service         # Auction lifecycle management
│   └── bidding-service         # WebSocket + bid processing service
├── deployments
├── internal
│   ├── api                     # HTTP handlers and middleware
│   │   ├── handlers
│   │   └── middleware
│   ├── config                  # Configuration management
│   ├── domain                  # Business entities and interfaces
│   │   └── repositories
│   ├── infrastructure          # External dependencies (Redis, MySQL, WebSocket)
│   │   ├── leader
│   │   ├── mysql
│   │   ├── redis
│   │   └── websocket
│   └── services                # Business logic layer
├── pkg                         # Shared utilities
│   ├── logger
│   └── utils
├── scripts                     # Database migrations and utilities
└── web                         # For webUI
```


## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_HOST` | Server bind address | `0.0.0.0` |
| `SERVER_PORT` | Server port | `8080` |
| `REDIS_ADDRESS` | Redis connection string | `localhost:6379` |
| `REDIS_PASSWORD` | Redis password | `` |
| `REDIS_DB` | Redis database number | `0` |
| `MYSQL_DSN` | MySQL connection string | See config.yaml |
| `INSTANCE_ID` | Unique instance identifier | `auction-service-1` |

## Performance Considerations

### Low Latency Optimizations
- **Local cache validation** before Redis calls
- **Single Redis operation** for bid processing using Lua scripts
- **Async MySQL writes** to avoid blocking bid processing
- **WebSocket-only** communication for real-time updates

### Scalability Features
- **Stateless services** - can run multiple instances
- **Event-driven architecture** - loose coupling between components
- **Leader election** - prevents race conditions in auction management
- **Redis pub/sub** - efficient cross-node communication

### Fault Tolerance
- **Health checks** for service discovery
- **Graceful shutdown** with proper cleanup
- **Leader failover** with automatic re-election
- **Connection cleanup** on service failures

## Troubleshooting

### Common Issues

1. **Services won't start**
   ```bash
   # Check if Redis and MySQL are running
   docker-compose -f deployments/docker-compose.yml ps
   
   # Check logs
   make docker-logs
   ```

2. **WebSocket connection fails**
   ```bash
   # Verify auction service is running
   curl http://localhost:8080/health
   
   # Check if auction exists
   curl http://localhost:8081/api/v1/auctions/{auction_id}
   ```

3. **Bids not processing**
   ```bash
   # Check Redis connectivity
   redis-cli ping
   
   # Check bid validation rules
   redis-cli GET bid_validation_rules
   ```

### Debug Mode

Run services with debug logging:
```bash
LOG_LEVEL=debug make run-auction-service
```

### Monitoring

Check service health:
```bash
# Auction Service
curl http://localhost:8080/health

# Auction Manager  
curl http://localhost:8081/health
```
 