package leader

import (
	_ "auction-system/internal/domain"
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisLeaderElection struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisLeaderElection(client *redis.Client, ttl time.Duration) *RedisLeaderElection {
	return &RedisLeaderElection{
		client: client,
		ttl:    ttl,
	}
}

func (r *RedisLeaderElection) BecomeLeader(ctx context.Context, instanceID string) (bool, error) {
	result, err := r.client.SetNX(ctx, "auction_leader", instanceID, r.ttl).Result()
	if err != nil {
		return false, err
	}

	if result {
		// Start heartbeat to maintain leadership
		go r.maintainLeadership(instanceID)
	}

	return result, nil
}

func (r *RedisLeaderElection) IsLeader(ctx context.Context, instanceID string) (bool, error) {
	currentLeader, err := r.client.Get(ctx, "auction_leader").Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	return currentLeader == instanceID, nil
}

func (r *RedisLeaderElection) ReleaseLeadership(ctx context.Context, instanceID string) error {
	// Use Lua script to ensure atomic release
	luaScript := `
        if redis.call("GET", KEYS[1]) == ARGV[1] then
            return redis.call("DEL", KEYS[1])
        else
            return 0
        end
    `

	_, err := r.client.Eval(ctx, luaScript, []string{"auction_leader"}, instanceID).Result()
	return err
}

func (r *RedisLeaderElection) maintainLeadership(instanceID string) {
	ticker := time.NewTicker(r.ttl / 3) // Refresh at 1/3 of TTL
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		// Extend TTL if still leader
		luaScript := `
            if redis.call("GET", KEYS[1]) == ARGV[1] then
                return redis.call("EXPIRE", KEYS[1], ARGV[2])
            else
                return 0
            end
        `

		result, err := r.client.Eval(ctx, luaScript, []string{"auction_leader"},
			instanceID, int(r.ttl.Seconds())).Result()

		cancel()

		if err != nil || result.(int64) == 0 {
			// Lost leadership, stop heartbeat
			break
		}
	}
}
