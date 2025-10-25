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
	NoLimit    cacheLimit    = -1
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
	DefaultCheckInterval time.Duration = 5000
	DefaultCacheLimit    cacheLimit    = 2000

	ErrCacheNotFound = fmt.Errorf("cache not found")
	ErrCacheExpired  = fmt.Errorf("cache expired")
)

type cacheOption int
type cacheLimit = int

type cacheIter interface {
	GetItems() [][]byte
}

type keeperIter interface {
	Add(cache cacheIter)
	GetItems() [][][]byte
}

type CacheKeeper struct {
	Caches []cacheIter
	mu     sync.Mutex
}

func NewKeeper() *CacheKeeper {
	return &CacheKeeper{}
}

func (ck *CacheKeeper) Add(cache cacheIter) {
	ck.mu.Lock()
	ck.Caches = append(ck.Caches, cache)
	ck.mu.Unlock()
}

func (ck *CacheKeeper) GetItems() (items [][][]byte) {
	ck.mu.Lock()
	defer ck.mu.Unlock()

	for _, cache := range ck.Caches {
		items = append(items, cache.GetItems())
	}

	return nil
}

type cacheItem[T any] struct {
	Err      error
	Expires  time.Time
	Duration time.Duration
	Value    T
}

type Cache[T any] struct {
	Items     map[string]cacheItem[T]
	mu        sync.RWMutex
	Ctx       context.Context
	lastCheck time.Time
	Limit     int
}

func NewCache[T any](ctx context.Context, cleanup ...bool) *Cache[T] {
	c := &Cache[T]{Items: map[string]cacheItem[T]{}, Ctx: ctx, Limit: DefaultCacheLimit}

	if len(cleanup) == 0 || cleanup[0] {
		c.cleanupTask(1 * time.Minute)
	}

	return c
}

func (c *Cache[T]) cleanupTask(t time.Duration) *Cache[T] {
	go func() {
		for {
			select {
			case <-c.Ctx.Done():
				return
			case <-time.After(t):
				c.Check()
			}
		}
	}()
	return c
}

func (c *Cache[T]) SetLimit(limit int) *Cache[T] {
	c.Limit = limit
	return c
}

func (c *Cache[T]) Nolimit() *Cache[T] {
	c.Limit = NoLimit
	return c
}

func (c *Cache[T]) Keeper(keeper keeperIter) *Cache[T] {
	keeper.Add(c)
	return c
}

func (c *Cache[T]) Exists(key string, options ...cacheOption) bool {
	v, _ := c.GetErr(key, options...)
	return v != nil
}

func (c *Cache[T]) ExistsSet(key string, data T, expires time.Duration, options ...cacheOption) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, err := c.getUnsafe(key, options...)
	if err == nil {
		return true
	}

	c.setUnsafe(key, &data, nil, expires)

	return false
}

func (c *Cache[T]) GetSet(key string, data T, expires time.Duration, options ...cacheOption) *T {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, err := c.getUnsafe(key, options...)
	if v != nil && err != nil {
		return nil
	} else if v != nil {
		return &v.Value
	}

	c.setUnsafe(key, &data, nil, expires)

	return nil
}

func (c *Cache[T]) Get(key string, options ...cacheOption) *T {
	v, err := c.GetErr(key, options...)
	if err != nil {
		return nil
	}

	return v
}

func (c *Cache[T]) GetErr(key string, options ...cacheOption) (*T, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, err := c.getUnsafe(key, options...)
	if item == nil && err != nil {
		return nil, err
	} else if err != nil {
		return &item.Value, err
	}

	return &item.Value, nil
}

func (c *Cache[T]) Set(key string, data T, expires ...time.Duration) error {
	return c.SetErr(key, &data, nil, expires...)
}

func (c *Cache[T]) SetErr(key string, data *T, err error, expires ...time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.setUnsafe(key, data, err, expires...)
}

func (c *Cache[T]) getUnsafe(key string, options ...cacheOption) (*cacheItem[T], error) {
	item, ok := c.Items[key]
	if !ok {
		return nil, ErrCacheNotFound
	}

	if item.Duration != NoExpire && item.Expires.Before(time.Now()) {
		delete(c.Items, key)
		return nil, ErrCacheExpired
	}

	if item.Err != nil {
		return &item, item.Err
	}

	if slices.Contains(options, ResetTimer) {
		item.Expires = time.Now().Add(item.Duration)
		c.Items[key] = item
	}

	c.checkUnsafe()

	return &item, nil
}

func (c *Cache[T]) setUnsafe(key string, data *T, err error, expires ...time.Duration) error {
	var holder T
	if data != nil {
		holder = *data
	}

	expiry := NoExpire
	if len(expires) > 0 {
		expiry = expires[0]
	}

	if item, ok := c.Items[key]; ok {
		if item.Duration == NoExpire || item.Duration > expiry {
			expiry = item.Duration
		}

		if item.Err != nil && err == nil {
			err = item.Err
		}
	}

	c.Items[key] = cacheItem[T]{Expires: time.Now().Add(expiry), Err: err, Duration: expiry, Value: holder}

	c.checkUnsafe()

	return err
}

func (c *Cache[T]) Remove(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, exists := c.Items[key]
	if exists {
		delete(c.Items, key)
	}

	return exists
}

func (c *Cache[T]) GetItems() [][]byte {
	c.mu.RLock()
	defer c.mu.RUnlock()

	items := make([][]byte, 0, len(c.Items))
	for _, v := range c.Items {
		bytes, _ := json.Marshal(v.Value)
		items = append(items, bytes)
	}

	return items
}

func (c *Cache[T]) Length() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.Items)
}

func (c *Cache[T]) checkUnsafe() {
	now := time.Now()

	if time.Since(c.lastCheck) < DefaultCheckInterval {
		return
	}

	c.lastCheck = now

	for key, item := range c.Items {
		if item.Duration != NoExpire && now.After(item.Expires) {
			delete(c.Items, key)
		}
	}

	if c.Limit != NoLimit {
		for len(c.Items) > c.Limit {
			for key := range c.Items {
				delete(c.Items, key)
				break
			}
		}
	}
}

func (c *Cache[T]) Check() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checkUnsafe()
}
