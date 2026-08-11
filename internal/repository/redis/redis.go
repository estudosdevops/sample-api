package redisrepo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/estudosdevops/sample-api/internal/domain"
	"github.com/estudosdevops/sample-api/internal/infra"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type RedisRepo struct {
	client *redis.Client
}

func NewRedis(addr string) (*RedisRepo, error) {
	opt := &redis.Options{Addr: addr}
	c := redis.NewClient(opt)
	return &RedisRepo{client: c}, nil
}

func (r *RedisRepo) Close() error {
	return r.client.Close()
}

// Ping checks connectivity to Redis.
func (r *RedisRepo) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// GetAddress returns a cached Address by cep, or ErrNotFound.
func (r *RedisRepo) GetAddress(ctx context.Context, cep string) (*domain.Address, error) {
	tracer := otel.Tracer("redis")
	ctx, span := tracer.Start(ctx, "Redis GET")
	span.SetAttributes(attribute.String("db.system", "redis"), attribute.String("redis.operation", "GET"), attribute.String("db.redis.key", cep))
	defer span.End()

	b, err := r.client.Get(ctx, cep).Bytes()
	if err != nil {
		if err == redis.Nil {
			infra.LoggerFromContext(ctx).Info("redis cache miss", "cep", cep)
			span.SetAttributes(attribute.String("cache.hit", "false"))
			return nil, domain.ErrNotFound
		}
		span.RecordError(err)
		return nil, fmt.Errorf("redis get: %w", err)
	}
	var a domain.Address
	if err := json.Unmarshal(b, &a); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("redis unmarshal: %w", err)
	}
	infra.LoggerFromContext(ctx).Info("redis cache hit", "cep", cep)
	span.SetAttributes(attribute.String("cache.hit", "true"))
	return &a, nil
}

// SetAddress caches an Address with given cep.
func (r *RedisRepo) SetAddress(ctx context.Context, a *domain.Address) error {
	tracer := otel.Tracer("redis")
	ctx, span := tracer.Start(ctx, "Redis SET")
	span.SetAttributes(attribute.String("db.system", "redis"), attribute.String("redis.operation", "SET"), attribute.String("db.redis.key", a.CEP))
	defer span.End()

	infra.LoggerFromContext(ctx).Info("redis cache set", "cep", a.CEP)

	b, err := json.Marshal(a)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("marshal: %w", err)
	}
	if err := r.client.Set(ctx, a.CEP, b, 0).Err(); err != nil {
		span.RecordError(err)
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}
