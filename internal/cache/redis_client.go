package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient wraps Redis operations for SecureStor
type RedisClient struct {
	client *redis.Client
	logger *log.Logger
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL        string
	Password   string
	DB         int
	MaxRetries int
	PoolSize   int
}

// NewRedisClient creates a new Redis client
func NewRedisClient(config RedisConfig, logger *log.Logger) (*RedisClient, error) {
	opts := &redis.Options{
		Addr:       config.URL,
		Password:   config.Password,
		DB:         config.DB,
		MaxRetries: config.MaxRetries,
		PoolSize:   config.PoolSize,
	}

	// Parse URL if provided
	if config.URL != "" {
		parsedOpts, err := redis.ParseURL(config.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
		}
		opts = parsedOpts

		// Override with specific config if provided
		if config.Password != "" {
			opts.Password = config.Password
		}
		if config.DB != 0 {
			opts.DB = config.DB
		}
		if config.MaxRetries > 0 {
			opts.MaxRetries = config.MaxRetries
		}
		if config.PoolSize > 0 {
			opts.PoolSize = config.PoolSize
		}
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Printf("Successfully connected to Redis at %s", opts.Addr)

	return &RedisClient{
		client: client,
		logger: logger,
	}, nil
}

// Set stores a value with expiration
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	err = r.client.Set(ctx, key, data, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}

	return nil
}

// Get retrieves and unmarshals a value
func (r *RedisClient) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrKeyNotFound
		}
		return fmt.Errorf("failed to get key %s: %w", key, err)
	}

	err = json.Unmarshal([]byte(data), dest)
	if err != nil {
		return fmt.Errorf("failed to unmarshal value for key %s: %w", key, err)
	}

	return nil
}

// Delete removes a key
func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	err := r.client.Del(ctx, keys...).Err()
	if err != nil {
		return fmt.Errorf("failed to delete keys: %w", err)
	}
	return nil
}

// Exists checks if a key exists
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence of key %s: %w", key, err)
	}
	return count > 0, nil
}

// SetExpiration sets expiration on an existing key
func (r *RedisClient) SetExpiration(ctx context.Context, key string, expiration time.Duration) error {
	err := r.client.Expire(ctx, key, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to set expiration for key %s: %w", key, err)
	}
	return nil
}

// Increment increments a numeric value
func (r *RedisClient) Increment(ctx context.Context, key string) (int64, error) {
	val, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment key %s: %w", key, err)
	}
	return val, nil
}

// IncrementBy increments a numeric value by a specific amount
func (r *RedisClient) IncrementBy(ctx context.Context, key string, value int64) (int64, error) {
	val, err := r.client.IncrBy(ctx, key, value).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment key %s by %d: %w", key, value, err)
	}
	return val, nil
}

// SetHash sets a field in a hash
func (r *RedisClient) SetHash(ctx context.Context, key, field string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	err = r.client.HSet(ctx, key, field, data).Err()
	if err != nil {
		return fmt.Errorf("failed to set hash field %s.%s: %w", key, field, err)
	}

	return nil
}

// GetHash gets a field from a hash
func (r *RedisClient) GetHash(ctx context.Context, key, field string, dest interface{}) error {
	data, err := r.client.HGet(ctx, key, field).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrKeyNotFound
		}
		return fmt.Errorf("failed to get hash field %s.%s: %w", key, field, err)
	}

	err = json.Unmarshal([]byte(data), dest)
	if err != nil {
		return fmt.Errorf("failed to unmarshal hash field value %s.%s: %w", key, field, err)
	}

	return nil
}

// GetAllHash gets all fields from a hash
func (r *RedisClient) GetAllHash(ctx context.Context, key string) (map[string]string, error) {
	data, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get all hash fields for %s: %w", key, err)
	}
	return data, nil
}

// AddToList adds an item to a list
func (r *RedisClient) AddToList(ctx context.Context, key string, values ...interface{}) error {
	jsonValues := make([]interface{}, len(values))
	for i, val := range values {
		data, err := json.Marshal(val)
		if err != nil {
			return fmt.Errorf("failed to marshal list item: %w", err)
		}
		jsonValues[i] = data
	}

	err := r.client.LPush(ctx, key, jsonValues...).Err()
	if err != nil {
		return fmt.Errorf("failed to add to list %s: %w", key, err)
	}

	return nil
}

// GetListRange gets items from a list by range
func (r *RedisClient) GetListRange(ctx context.Context, key string, start, stop int64, dest interface{}) error {
	data, err := r.client.LRange(ctx, key, start, stop).Result()
	if err != nil {
		return fmt.Errorf("failed to get list range %s: %w", key, err)
	}

	// Unmarshal each item in the slice
	return r.unmarshalSlice(data, dest)
}

// PublishMessage publishes a message to a channel
func (r *RedisClient) PublishMessage(ctx context.Context, channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = r.client.Publish(ctx, channel, data).Err()
	if err != nil {
		return fmt.Errorf("failed to publish to channel %s: %w", channel, err)
	}

	return nil
}

// Subscribe creates a subscription to a channel
func (r *RedisClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return r.client.Subscribe(ctx, channels...)
}

// FlushDB clears the current database (use with caution!)
func (r *RedisClient) FlushDB(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Health checks Redis connection health
func (r *RedisClient) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Keys returns all keys matching a pattern
func (r *RedisClient) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// GetJSON retrieves a JSON value and unmarshals it into an interface{}
func (r *RedisClient) GetJSON(ctx context.Context, key string) (interface{}, error) {
	data, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, err
	}

	return result, nil
}

// TTL returns the time-to-live of a key
func (r *RedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return ttl, nil
}

// Helper function to unmarshal slice data
func (r *RedisClient) unmarshalSlice(data []string, dest interface{}) error {
	jsonArray := "["
	for i, item := range data {
		if i > 0 {
			jsonArray += ","
		}
		jsonArray += item
	}
	jsonArray += "]"

	return json.Unmarshal([]byte(jsonArray), dest)
}

// Custom errors
var (
	ErrKeyNotFound = fmt.Errorf("key not found")
)
