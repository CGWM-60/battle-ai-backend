package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type ResponseCache struct {
	redis     *RedisService
	prefix    string
	items     sync.Map
	versions  sync.Map
	namespace sync.Mutex
	sets      atomic.Uint64
}

type cacheItem struct {
	value     string
	expiresAt time.Time
}

func NewResponseCache(redis *RedisService, prefix string) *ResponseCache {
	if prefix == "" {
		prefix = "app"
	}
	return &ResponseCache{redis: redis, prefix: prefix}
}

func (c *ResponseCache) GetJSON(ctx context.Context, namespace string, key string, dest any) bool {
	if c == nil || dest == nil {
		return false
	}

	cacheKey := c.cacheKey(ctx, namespace, key)
	if raw, ok := c.getLocal(cacheKey); ok {
		return json.Unmarshal([]byte(raw), dest) == nil
	}
	if c.redis == nil {
		return false
	}

	redisCtx, cancel := cacheRedisContext(ctx)
	defer cancel()
	raw, ok, err := c.redis.GetString(redisCtx, cacheKey)
	if err != nil || !ok {
		return false
	}
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		return false
	}
	c.items.Store(cacheKey, cacheItem{value: raw, expiresAt: time.Now().Add(time.Second)})
	return true
}

func (c *ResponseCache) SetJSON(ctx context.Context, namespace string, key string, value any, ttl time.Duration) {
	if c == nil || value == nil || ttl <= 0 {
		return
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	cacheKey := c.cacheKey(ctx, namespace, key)
	payload := string(raw)
	c.items.Store(cacheKey, cacheItem{value: payload, expiresAt: time.Now().Add(ttl)})
	if c.sets.Add(1)%256 == 0 {
		c.cleanupExpired()
	}
	if c.redis != nil {
		redisCtx, cancel := cacheRedisContext(ctx)
		defer cancel()
		_ = c.redis.SetString(redisCtx, cacheKey, payload, ttl)
	}
}

func (c *ResponseCache) InvalidateNamespace(ctx context.Context, namespace string) {
	if c == nil || namespace == "" {
		return
	}
	version := fmt.Sprintf("%d", time.Now().UnixNano())
	c.versions.Store(namespace, version)
	if c.redis != nil {
		redisCtx, cancel := cacheRedisContext(ctx)
		defer cancel()
		_ = c.redis.SetString(redisCtx, c.versionKey(namespace), version, 24*time.Hour)
	}
}

func (c *ResponseCache) cacheKey(ctx context.Context, namespace string, key string) string {
	return fmt.Sprintf("%s:%s:%s:%s", c.prefix, namespace, c.version(ctx, namespace), key)
}

func (c *ResponseCache) version(ctx context.Context, namespace string) string {
	if namespace == "" {
		namespace = "default"
	}
	if value, ok := c.versions.Load(namespace); ok {
		if version, ok := value.(string); ok && version != "" {
			return version
		}
	}

	c.namespace.Lock()
	defer c.namespace.Unlock()
	if value, ok := c.versions.Load(namespace); ok {
		if version, ok := value.(string); ok && version != "" {
			return version
		}
	}

	version := "0"
	if c.redis != nil {
		redisCtx, cancel := cacheRedisContext(ctx)
		defer cancel()
		if raw, ok, err := c.redis.GetString(redisCtx, c.versionKey(namespace)); err == nil && ok && raw != "" {
			version = raw
		}
	}
	c.versions.Store(namespace, version)
	return version
}

func cacheRedisContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, 75*time.Millisecond)
}

func (c *ResponseCache) versionKey(namespace string) string {
	return fmt.Sprintf("%s:%s:version", c.prefix, namespace)
}

func (c *ResponseCache) getLocal(key string) (string, bool) {
	value, ok := c.items.Load(key)
	if !ok {
		return "", false
	}
	item, ok := value.(cacheItem)
	if !ok {
		c.items.Delete(key)
		return "", false
	}
	if time.Now().After(item.expiresAt) {
		c.items.Delete(key)
		return "", false
	}
	return item.value, true
}

func (c *ResponseCache) cleanupExpired() {
	now := time.Now()
	c.items.Range(func(key any, value any) bool {
		item, ok := value.(cacheItem)
		if !ok || now.After(item.expiresAt) {
			c.items.Delete(key)
		}
		return true
	})
}
