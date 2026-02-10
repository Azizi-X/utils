package cache

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"time"
)

const (
	ResetTimer cacheOption = iota
	ResetTimerOnErr

	NoLimit           cacheLimit = -1
	DefaultCacheLimit cacheLimit = 2000

	NoExpire time.Duration = -1

	DefaultDuration   = 15 * time.Minute
	HourDuration      = 1 * time.Hour
	FullDayDuration   = 24 * time.Hour
	HalfDayDuration   = 12 * time.Hour
	ShortDuration     = 5 * time.Minute
	TenMinuteDuration = 10 * time.Minute
)

var (
	ErrCacheNotFound = errors.New("cache not found")
	ErrCacheExpired  = errors.New("cache expired")

	keeper = NewKeeper()
)

type cacheOption int
type cacheLimit = int

type Cache[T any] struct {
	Items map[string]cacheItem[T]
	mu    sync.RWMutex
	Ctx   context.Context
	Limit int
}

type CacheKeeper struct {
	Caches []cacheInter
	mu     sync.Mutex
}

type cacheItem[T any] struct {
	Err      error
	Expires  time.Time
	Duration time.Duration
	Value    T
}

type cacheInter interface {
	C() context.Context
	GetItems() [][]byte
	Check()
}

type keeperInter interface {
	Add(cache cacheInter)
	GetItems() [][][]byte
}

func NewKeeper() *CacheKeeper {
	return &CacheKeeper{}
}

func (ck *CacheKeeper) Add(cache cacheInter) {
	ck.mu.Lock()
	ck.Caches = append(ck.Caches, cache)
	ck.mu.Unlock()
}

func (ck *CacheKeeper) GetItems() (items [][][]byte) {
	ck.mu.Lock()
	for _, cache := range ck.Caches {
		items = append(items, cache.GetItems())
	}
	ck.mu.Unlock()

	return
}

func (ck *CacheKeeper) _cleanup() {
	ticker := time.NewTicker(60 * time.Second)

	for range ticker.C {
		ck.mu.Lock()
		ck.Caches = slices.DeleteFunc(ck.Caches, func(cache cacheInter) bool {
			ctx := cache.C()
			return ctx != nil && ctx.Err() != nil
		})
		caches := slices.Clone(ck.Caches)
		ck.mu.Unlock()

		for _, cache := range caches {
			cache.Check()
		}
	}
}

func NewCache[T any](ctx context.Context) *Cache[T] {
	c := &Cache[T]{Items: map[string]cacheItem[T]{}, Ctx: ctx, Limit: DefaultCacheLimit}

	keeper.Add(c)

	return c
}

func (c *Cache[T]) C() context.Context {
	return c.Ctx
}

func (c *Cache[T]) SetLimit(limit int) *Cache[T] {
	c.Limit = limit
	return c
}

func (c *Cache[T]) Nolimit() *Cache[T] {
	c.Limit = NoLimit
	return c
}

func (c *Cache[T]) Keeper(keeper keeperInter) *Cache[T] {
	keeper.Add(c)
	return c
}

func (c *Cache[T]) Exists(key string, options ...cacheOption) bool {
	v, _ := c.GetErr(key, options...)
	return v != nil
}

func (c *Cache[T]) UniqueSet(key string, data T, expires time.Duration, options ...cacheOption) bool {
	c.mu.Lock()
	_, already_set := c.getSetUnsafe(key, data, expires, options...)
	c.mu.Unlock()
	return !already_set
}

func (c *Cache[T]) GetSet(key string, data T, expires time.Duration, options ...cacheOption) *T {
	c.mu.Lock()
	v, _ := c.getSetUnsafe(key, data, expires, options...)
	c.mu.Unlock()
	return v
}

func (c *Cache[T]) getSetUnsafe(key string, data T, expires time.Duration, options ...cacheOption) (*T, bool) {
	v, err := c.getUnsafe(key, options...)
	if v != nil && err != nil {
		return nil, false
	} else if v != nil {
		return &v.Value, true
	}

	c.setUnsafe(key, &data, nil, expires)

	return &data, false
}

func (c *Cache[T]) Get(key string, options ...cacheOption) *T {
	v, err := c.GetErr(key, options...)
	if err != nil {
		return nil
	}

	return v
}

func (c *Cache[T]) GetErr(key string, options ...cacheOption) (*T, error) {
	full_mu := slices.Contains(options, ResetTimer)

	if full_mu {
		c.mu.Lock()
	} else {
		c.mu.RLock()
	}

	item, err := c.getUnsafe(key, options...)

	if full_mu {
		c.mu.Unlock()
	} else {
		c.mu.RUnlock()
	}

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
	c.setUnsafe(key, data, err, expires...)
	c.mu.Unlock()

	return err
}

func (c *Cache[T]) getUnsafe(key string, options ...cacheOption) (*cacheItem[T], error) {
	item, ok := c.Items[key]
	if !ok {
		return nil, ErrCacheNotFound
	}

	if item.Duration != NoExpire && item.Expires.Before(time.Now()) {
		return nil, ErrCacheExpired
	}

	if (slices.Contains(options, ResetTimer) && item.Err == nil) ||
		(item.Err != nil && slices.Contains(options, ResetTimerOnErr)) {
		item.Expires = time.Now().Add(item.Duration)
		c.Items[key] = item
	}

	return &item, item.Err
}

func (c *Cache[T]) setUnsafe(key string, data *T, err error, expires ...time.Duration) {
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
	}

	c.Items[key] = cacheItem[T]{Expires: time.Now().Add(expiry), Err: err, Duration: expiry, Value: holder}
}

func (c *Cache[T]) Clear() {
	c.mu.Lock()
	c.Items = make(map[string]cacheItem[T])
	c.mu.Unlock()
}

func (c *Cache[T]) Remove(key string) bool {
	c.mu.Lock()
	_, ok := c.Items[key]
	if ok {
		delete(c.Items, key)
	}
	c.mu.Unlock()
	return ok
}

func (c *Cache[T]) GetItems() [][]byte {
	c.mu.RLock()

	items := make([][]byte, 0, len(c.Items))
	for _, v := range c.Items {
		bytes, _ := json.Marshal(v.Value)
		items = append(items, bytes)
	}

	c.mu.RUnlock()

	return items
}

func (c *Cache[T]) Length() int {
	c.mu.RLock()
	length := len(c.Items)
	c.mu.RUnlock()
	return length
}

func (c *Cache[T]) Check() {
	c.mu.Lock()
	now := time.Now()

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

	c.mu.Unlock()
}

func init() {
	go keeper._cleanup()
}
