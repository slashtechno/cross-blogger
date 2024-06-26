package internal

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitializeDb(name string, options *redis.Options) error {
	RedisClient = redis.NewClient(options)
	// Ping
	ctx := context.Background()
	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		return err
	}

	// --- FOR DEBUGGING ---
	err = RedisClient.Set(ctx, "key", "value", 0).Err()
	if err != nil {
		return err
	}
	val, err := RedisClient.Get(ctx, "key").Result()
	if err != nil {
		return err
	}
	log.Info("", "key", val)
	// --- FOR DEBUGGING ---

	return nil

}

// Initializes Redis options from configuration.
// Returns nil if Redis is disabled or on error, with an error explaining the failure.
func InitializeRedisOptions(viperMap map[string]interface{}) (*redis.Options, error) {
	if enabled, ok := viperMap["enable"].(bool); !ok || !enabled {
		// Explicitly return nil to indicate Redis is disabled or the enable flag is not properly set.
		return nil, nil
	}

	urlString, ok := viperMap["url"].(string)
	if !ok {
		// Handle the case where the URL is not a string or not present.
		return nil, fmt.Errorf("redis URL is not a string or missing")
	}

	opt, err := redis.ParseURL(urlString)
	if err != nil {
		// Wrap the error for additional context.
		return nil, fmt.Errorf("error parsing Redis URL: %w", err)
	}

	return opt, nil
}
