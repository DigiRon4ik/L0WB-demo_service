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

type Cache struct {
	capacity int
	cache    map[string]*list.Element
	queue    *list.List
	mu       sync.RWMutex
}

func NewCahce(capacity int) *Cache {
	return &Cache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		queue:    list.New(),
	}
}

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

func (c *Cache) Get(key string) (models.Order, bool) {
	c.mu.RLock()
	element, exists := c.cache[key]
	c.mu.RUnlock() // Освободить блокировку чтения

	if !exists {
		return models.Order{}, false
	}

	// Для изменения порядка очереди требуется блокировка на запись
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
