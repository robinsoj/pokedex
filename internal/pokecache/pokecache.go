package pokecache

import (
	"sync"
	"time"
)

type cacheEntry struct {
	CreatedAt time.Time
	Val       []byte
}

type Cache struct {
	Data     map[string]cacheEntry
	Mu       sync.Mutex
	Interval time.Duration
}

func NewCache(inteval time.Duration) *Cache {
	var cache Cache
	cache.Data = make(map[string]cacheEntry)
	cache.Interval = inteval
	go cache.ReadLoop()
	return &cache
}

func (c *Cache) Add(key string, value []byte) error {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	var ceValue cacheEntry
	ceValue.CreatedAt = time.Now()
	ceValue.Val = value
	c.Data[key] = ceValue

	return nil
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	entry, ok := c.Data[key]
	if !ok {
		return nil, false
	}
	return entry.Val, true
}

func (c *Cache) ReadLoop() {
	for {
		time.Sleep(c.Interval)
		c.Mu.Lock()
		for key, entry := range c.Data {
			if time.Since(entry.CreatedAt) > c.Interval {
				delete(c.Data, key)
			}
		}
		c.Mu.Unlock()
	}
}
