package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"
)

const (
	ResetTimer cacheOption   = 0
	NoExpire   time.Duration = -1

	DefaultDuration   = 15 * time.Minute
	HourDuration      = 1 * time.Hour
	FullDayDuration   = 24 * time.Hour
	HalfDayDuration   = 12 * time.Hour
	ShortDuration     = 5 * time.Minute
	TenMinuteDuration = 10 * time.Minute
)

var (
	SleepTimer = 2 * time.Second
	MaxEntries = 2000
)

type cacheOption int

type cacheItem struct {
	Bytes    []byte
	Err      error
	Expires  time.Time
	Duration time.Duration
	Any      any
}

type Cache struct {
	Items map[string]cacheItem
	mu    sync.Mutex
	Ctx   context.Context
}

func NewCache(ctx context.Context) *Cache {
	cache := &Cache{Items: map[string]cacheItem{}, Ctx: ctx}

	go cache.loop()

	return cache
}

func (c *Cache) Exists(key string, options ...cacheOption) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.Items[key]
	if !ok {
		return false
	}

	if item.Duration != NoExpire && item.Expires.Before(time.Now()) {
		delete(c.Items, key)
		return false
	}

	if slices.Contains(options, ResetTimer) {
		item.Expires = time.Now().Add(item.Duration)
	}

	return true
}

func (c *Cache) ExistsSet(key string, data any, expires time.Duration, options ...cacheOption) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, err := c.getItemUnsafe(key, options...)
	if err == nil {
		return true
	}

	c.setUnsafe(key, data, nil, expires)

	return false
}

func (c *Cache) GetItems() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	items := make([][]byte, 0, len(c.Items))
	for _, v := range c.Items {
		items = append(items, v.Bytes)
	}

	return items
}

func (c *Cache) GetAny(key string, options ...cacheOption) (any, bool) {
	v, ok, err := c.GetAnyErr(key, options...)
	if err != nil {
		return nil, false
	}

	return v, ok
}

func (c *Cache) GetAnyErr(key string, options ...cacheOption) (any, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.Items[key]
	if !ok {
		return nil, false, nil
	}

	if item.Duration != NoExpire && item.Expires.Before(time.Now()) {
		delete(c.Items, key)
		return nil, false, nil
	}

	if item.Err != nil {
		return nil, true, item.Err
	}

	if slices.Contains(options, ResetTimer) {
		item.Expires = time.Now().Add(item.Duration)
		c.Items[key] = item
	}

	return item.Any, true, nil
}

func (c *Cache) getItemUnsafe(key string, options ...cacheOption) (*cacheItem, error) {
	item, ok := c.Items[key]
	if !ok {
		return nil, fmt.Errorf("cache not found")
	}

	if item.Duration != NoExpire && item.Expires.Before(time.Now()) {
		delete(c.Items, key)
		return nil, fmt.Errorf("cache expired")
	}

	if item.Err != nil {
		return nil, item.Err
	}

	if slices.Contains(options, ResetTimer) {
		item.Expires = time.Now().Add(item.Duration)
		c.Items[key] = item
	}

	return &item, nil
}

func (c *Cache) Get(key string, v any, options ...cacheOption) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, err := c.getItemUnsafe(key, options...)
	if err != nil {
		return false, err
	}

	return true, json.Unmarshal(item.Bytes, v)
}

func (c *Cache) Set(key string, data any, expires ...time.Duration) error {
	return c.SetErr(key, data, nil, expires...)
}

func (c *Cache) SetErr(key string, data any, err error, expires ...time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.setUnsafe(key, data, err, expires...)
}

func (c *Cache) setUnsafe(key string, data any, err error, expires ...time.Duration) error {
	bytes, jsonErr := json.Marshal(data)
	if jsonErr != nil {
		return jsonErr
	}

	expiry := NoExpire
	if len(expires) > 0 {
		expiry = expires[0]
	}

	c.Items[key] = cacheItem{Bytes: bytes, Expires: time.Now().Add(expiry), Err: err, Duration: expiry}

	return err
}

func (c *Cache) SetAny(key string, data any, expires ...time.Duration) error {
	return c.SetAnyErr(key, data, nil, expires...)
}

func (c *Cache) SetAnyErr(key string, data any, err error, expires ...time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiry := NoExpire
	if len(expires) > 0 {
		expiry = expires[0]
	}

	c.Items[key] = cacheItem{Expires: time.Now().Add(expiry), Err: err, Duration: expiry, Any: data}

	return err
}

func (c *Cache) Remove(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, exists := c.Items[key]
	if exists {
		delete(c.Items, key)
	}

	return exists
}

func (c *Cache) Check() {
	c.mu.Lock()
	for key, item := range c.Items {
		if item.Duration != NoExpire && item.Expires.Before(time.Now()) {
			delete(c.Items, key)
		}
	}
	for len(c.Items) > MaxEntries {
		for key := range c.Items {
			delete(c.Items, key)
			break
		}
	}
	c.mu.Unlock()
}

func (c *Cache) loop() {
	for c.Ctx.Err() == nil {
		time.Sleep(SleepTimer)
		c.Check()
	}
}
