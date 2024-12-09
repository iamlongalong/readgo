package readgo

import (
	"sync"
	"time"
)

// Cache provides a simple in-memory cache for type information
type Cache struct {
	mu    sync.RWMutex
	types map[TypeCacheKey]*TypeInfo
	hits  int64
	ttl   time.Duration
}

// TypeCacheKey is the key used for caching type information
type TypeCacheKey struct {
	Package  string
	TypeName string
	Kind     string
}

// NewCache creates a new cache with the given TTL
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		types: make(map[TypeCacheKey]*TypeInfo),
		ttl:   ttl,
	}
}

// GetType retrieves a type from the cache
func (c *Cache) GetType(key TypeCacheKey) (*TypeInfo, bool) {
	if c == nil || c.ttl <= 0 {
		return nil, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if info, ok := c.types[key]; ok {
		c.hits++
		return info, true
	}
	return nil, false
}

// SetType stores a type in the cache
func (c *Cache) SetType(key TypeCacheKey, info *TypeInfo) {
	if c == nil || c.ttl <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.types[key] = info
}

// Stats returns cache statistics
func (c *Cache) Stats() map[string]interface{} {
	if c == nil {
		return map[string]interface{}{
			"hits":    int64(0),
			"entries": int64(0),
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"hits":    c.hits,
		"entries": int64(len(c.types)),
	}
}
