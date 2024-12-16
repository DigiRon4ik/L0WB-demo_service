// Package cache provides an in-memory cache implementation with a least-recently-used (LRU)
// eviction policy.
//
// The Cache stores orders and ensures that the most recently accessed or added orders are kept in
// memory, while evicting the least recently used orders
// when the cache reaches its specified capacity.
//
// The cache is thread-safe, supporting concurrent access and modification through proper
// synchronization mechanisms.
package cache

import (
	"container/list"
	"demo_service/internal/models"
	"sync"
)

type cacheItem struct {
	Key   string
	Value models.Order
}

// Cache represents an in-memory cache with a specified capacity,
// a mapping of keys to list elements, and a queue for maintaining the order of items.
type Cache struct {
	capacity int
	cache    map[string]*list.Element
	queue    *list.List
	mu       sync.RWMutex
}

// New creates and returns a new Cache instance with the specified capacity.
func New(capacity int) *Cache {
	return &Cache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		queue:    list.New(),
	}
}

// Set adds or updates an order in the cache, moves it to the front of the queue,
// and evicts the least recently used item if the cache is at capacity.
func (c *Cache) Set(key string, value models.Order) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, exists := c.cache[key]; exists {
		c.queue.MoveToFront(element)
		element.Value.(*cacheItem).Value = value
		return true
	}

	if c.queue.Len() == c.capacity {
		c.purge()
	}

	cacheItem := &cacheItem{
		Key:   key,
		Value: value,
	}

	element := c.queue.PushFront(cacheItem)
	c.cache[key] = element

	return true
}

// Get retrieves an order from the cache by its key, moves it to the front of the queue,
// and returns the order along with a boolean indicating its existence.
func (c *Cache) Get(key string) (models.Order, bool) {
	c.mu.RLock()
	element, exists := c.cache[key]
	c.mu.RUnlock() // Release read lock

	if !exists {
		return models.Order{}, false
	}

	// A write lock is required to change the order of the queue
	c.mu.Lock()
	defer c.mu.Unlock()

	c.queue.MoveToFront(element)
	return element.Value.(*cacheItem).Value, true
}

func (c *Cache) purge() {
	if element := c.queue.Back(); element != nil {
		cacheItem := c.queue.Remove(element).(*cacheItem)
		delete(c.cache, cacheItem.Key)
	}
}
