package cache

import (
	"sync"
	"time"
)

type item[T any] struct {
	value     T
	expiresAt time.Time
}

func (i *item[T]) isExpired() bool {
	return time.Now().After(i.expiresAt)
}

type Cache[K comparable, V any] struct {
	mu    sync.RWMutex
	items map[K]*item[V]
	ttl   time.Duration
}

func New[K comparable, V any](ttl time.Duration) *Cache[K, V] {
	c := &Cache[K, V]{
		items: make(map[K]*item[V]),
		ttl:   ttl,
	}

	go c.startCleanup()

	return c
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok || item.isExpired() {
		var zero V
		return zero, false
	}

	return item.value, true
}

func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &item[V]{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

func (c *Cache[K, V]) startCleanup() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

func (c *Cache[K, V]) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, item := range c.items {
		if item.isExpired() {
			delete(c.items, key)
		}
	}
}
