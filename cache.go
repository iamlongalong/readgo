package readgo

import (
	"sync"
	"time"
)

// Cache keys
type (
	// TypeCacheKey represents a key for type lookup cache
	TypeCacheKey struct {
		Package  string
		TypeName string
		Kind     string // empty or "interface"
	}

	// PackageCacheKey represents a key for package analysis cache
	PackageCacheKey struct {
		Path string
		Mode string // "full" or "types-only"
	}

	// FileCacheKey represents a key for file analysis cache
	FileCacheKey struct {
		Path    string
		ModTime time.Time
	}
)

// CacheEntry represents a cached analysis result
type CacheEntry struct {
	Data      interface{}
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Cache implements a thread-safe cache for analysis results
type Cache struct {
	typeCache    sync.Map // map[TypeCacheKey]CacheEntry
	packageCache sync.Map // map[PackageCacheKey]CacheEntry
	fileCache    sync.Map // map[FileCacheKey]CacheEntry
	ttl          time.Duration
}

// NewCache creates a new cache instance with the specified TTL
func NewCache(ttl time.Duration) *Cache {
	return &Cache{ttl: ttl}
}

// GetType retrieves a cached type lookup result
func (c *Cache) GetType(key TypeCacheKey) (*TypeInfo, bool) {
	if entry, ok := c.typeCache.Load(key); ok {
		cacheEntry := entry.(CacheEntry)
		if time.Now().Before(cacheEntry.ExpiresAt) {
			return cacheEntry.Data.(*TypeInfo), true
		}
		c.typeCache.Delete(key)
	}
	return nil, false
}

// SetType caches a type lookup result
func (c *Cache) SetType(key TypeCacheKey, value *TypeInfo) {
	now := time.Now()
	c.typeCache.Store(key, CacheEntry{
		Data:      value,
		CreatedAt: now,
		ExpiresAt: now.Add(c.ttl),
	})
}

// GetPackage retrieves a cached package analysis result
func (c *Cache) GetPackage(key PackageCacheKey) (*AnalysisResult, bool) {
	if entry, ok := c.packageCache.Load(key); ok {
		cacheEntry := entry.(CacheEntry)
		if time.Now().Before(cacheEntry.ExpiresAt) {
			return cacheEntry.Data.(*AnalysisResult), true
		}
		c.packageCache.Delete(key)
	}
	return nil, false
}

// SetPackage caches a package analysis result
func (c *Cache) SetPackage(key PackageCacheKey, value *AnalysisResult) {
	now := time.Now()
	c.packageCache.Store(key, CacheEntry{
		Data:      value,
		CreatedAt: now,
		ExpiresAt: now.Add(c.ttl),
	})
}

// GetFile retrieves a cached file analysis result
func (c *Cache) GetFile(key FileCacheKey) (*AnalysisResult, bool) {
	if entry, ok := c.fileCache.Load(key); ok {
		cacheEntry := entry.(CacheEntry)
		if time.Now().Before(cacheEntry.ExpiresAt) {
			return cacheEntry.Data.(*AnalysisResult), true
		}
		c.fileCache.Delete(key)
	}
	return nil, false
}

// SetFile caches a file analysis result
func (c *Cache) SetFile(key FileCacheKey, value *AnalysisResult) {
	now := time.Now()
	c.fileCache.Store(key, CacheEntry{
		Data:      value,
		CreatedAt: now,
		ExpiresAt: now.Add(c.ttl),
	})
}

// Clear clears all cached entries
func (c *Cache) Clear() {
	c.typeCache = sync.Map{}
	c.packageCache = sync.Map{}
	c.fileCache = sync.Map{}
}

// Stats returns cache statistics
func (c *Cache) Stats() map[string]interface{} {
	stats := make(map[string]interface{})

	var typeCount, packageCount, fileCount int
	c.typeCache.Range(func(key, value interface{}) bool {
		typeCount++
		return true
	})
	c.packageCache.Range(func(key, value interface{}) bool {
		packageCount++
		return true
	})
	c.fileCache.Range(func(key, value interface{}) bool {
		fileCount++
		return true
	})

	stats["type_entries"] = typeCount
	stats["package_entries"] = packageCount
	stats["file_entries"] = fileCount
	stats["ttl_seconds"] = c.ttl.Seconds()

	return stats
}
