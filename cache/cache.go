package cache

import (
	"context"
	"encoding/json"
	"slices"
	"sync"
	"time"
)

const (
	ResetTimer Flag = 1 << iota
	ResetTimerOnErr
	ResetTimerWith

	NoLimit           = -1
	DefaultCacheLimit = 2000

	NoExpiration time.Duration = -1

	DefaultDuration   = 15 * time.Minute
	HalfHourDuration  = 30 * time.Minute
	HourDuration      = time.Hour
	DayDuration       = 24 * time.Hour
	WeekDuration      = 7 * DayDuration
	HalfDayDuration   = 12 * time.Hour
	TenMinuteDuration = 10 * time.Minute
	ShortDuration     = 5 * time.Minute
)

type Flag uint8

type Result[V any] struct {
	Value V
	Err   error
}

type Cache[K comparable, V any] struct {
	mu    sync.RWMutex
	items map[K]item[Result[V]]
	ctx   context.Context
	limit int
}

type item[V any] struct {
	value     V
	expiresAt int64
	duration  time.Duration
}

type managedCache interface {
	context() context.Context
	cleanup(int64)
	appendItems([][]byte, int64) [][]byte
}

type cacheList struct {
	mu     sync.Mutex
	caches []managedCache
}

type CacheKeeper struct {
	cacheList
}

var cleanupCaches cacheList

func New[K comparable, V any](ctx context.Context) *Cache[K, V] {
	cache := &Cache[K, V]{
		items: make(map[K]item[Result[V]]),
		ctx:   ctx,
		limit: DefaultCacheLimit,
	}

	cleanupCaches.add(cache)
	return cache
}

func NewKeeper() *CacheKeeper {
	return &CacheKeeper{}
}

func (l *cacheList) add(cache managedCache) {
	l.mu.Lock()
	l.caches = append(l.caches, cache)
	l.mu.Unlock()
}

func (l *cacheList) active() []managedCache {
	l.mu.Lock()
	l.caches = slices.DeleteFunc(l.caches, func(cache managedCache) bool {
		return cache.context().Err() != nil
	})
	active := slices.Clone(l.caches)
	l.mu.Unlock()
	return active
}

func (c *Cache[K, V]) Keeper(keeper *CacheKeeper) *Cache[K, V] {
	if keeper != nil {
		keeper.add(c)
	}
	return c
}

func (k *CacheKeeper) GetItems() (items [][]byte) {
	now := time.Now().UnixNano()
	for _, cache := range k.active() {
		items = cache.appendItems(items, now)
	}
	return
}

func runCleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for now := range ticker.C {
		timestamp := now.UnixNano()
		for _, cache := range cleanupCaches.active() {
			cache.cleanup(timestamp)
		}
	}
}

func (c *Cache[K, V]) SetLimit(limit int) *Cache[K, V] {
	if limit < 0 {
		limit = NoLimit
	}

	c.mu.Lock()
	c.limit = limit
	c.trimUnsafe()
	c.mu.Unlock()
	return c
}

func (c *Cache[K, V]) NoLimit() *Cache[K, V] {
	return c.SetLimit(NoLimit)
}

func (c *Cache[K, V]) Get(key K, flags ...Flag) (Result[V], bool) {
	flag := joinFlags(flags)
	if flag != 0 {
		c.mu.Lock()
		value, found := c.getUnsafe(key, flag, 0)
		c.mu.Unlock()
		return value, found
	}

	c.mu.RLock()
	entry, found := c.items[key]
	if found && entry.expiresAt > 0 && time.Now().UnixNano() >= entry.expiresAt {
		found = false
	}
	c.mu.RUnlock()

	if !found {
		var zero Result[V]
		return zero, false
	}
	return entry.value, true
}

func (c *Cache[K, V]) Exists(key K, flags ...Flag) bool {
	_, found := c.Get(key, flags...)
	return found
}

func (c *Cache[K, V]) Set(key K, value V) {
	c.SetFor(key, value, NoExpiration)
}

func (c *Cache[K, V]) SetFor(key K, value V, duration time.Duration) {
	c.setFor(key, Result[V]{Value: value}, duration)
}

func (c *Cache[K, V]) SetErr(key K, err error) {
	c.SetErrFor(key, err, NoExpiration)
}

func (c *Cache[K, V]) SetErrFor(key K, err error, duration time.Duration) {
	c.setFor(key, Result[V]{Err: err}, duration)
}

func (c *Cache[K, V]) setFor(key K, value Result[V], duration time.Duration) {
	c.mu.Lock()
	c.setUnsafe(key, value, duration)
	c.mu.Unlock()
}

func (c *Cache[K, V]) GetSet(key K, value V, flags ...Flag) (Result[V], bool) {
	result, found, _ := c.getSet(key, value, NoExpiration, joinFlags(flags))
	return result, found
}

func (c *Cache[K, V]) GetSetFor(key K, value V, duration time.Duration, flags ...Flag) (Result[V], bool) {
	result, found, _ := c.getSet(key, value, duration, joinFlags(flags))
	return result, found
}

func (c *Cache[K, V]) GetSetFn(key K, fn func() V, flags ...Flag) (Result[V], bool) {
	return c.getSetFn(key, fn, NoExpiration, joinFlags(flags))
}

func (c *Cache[K, V]) GetSetFnFor(key K, fn func() V, duration time.Duration, flags ...Flag) (Result[V], bool) {
	return c.getSetFn(key, fn, duration, joinFlags(flags))
}

func (c *Cache[K, V]) SetUnique(key K, value V, flags ...Flag) bool {
	_, found, stored := c.getSet(key, value, NoExpiration, joinFlags(flags))
	return !found && stored
}

func (c *Cache[K, V]) SetUniqueFor(key K, value V, duration time.Duration, flags ...Flag) bool {
	_, found, stored := c.getSet(key, value, duration, joinFlags(flags))
	return !found && stored
}

func (c *Cache[K, V]) getSet(key K, value V, duration time.Duration, flag Flag) (Result[V], bool, bool) {
	c.mu.Lock()
	current, found := c.getUnsafe(key, flag, duration)
	if found {
		c.mu.Unlock()
		return current, true, false
	}

	result := Result[V]{Value: value}
	stored := c.setUnsafe(key, result, duration)
	c.mu.Unlock()
	return result, false, stored
}

func (c *Cache[K, V]) getSetFn(key K, fn func() V, duration time.Duration, flag Flag) (Result[V], bool) {
	c.mu.Lock()
	current, found := c.getUnsafe(key, flag, duration)
	if found {
		c.mu.Unlock()
		return current, true
	}

	result := Result[V]{Value: fn()}
	c.setUnsafe(key, result, duration)
	c.mu.Unlock()
	return result, false
}

func (c *Cache[K, V]) getUnsafe(key K, flag Flag, duration time.Duration) (Result[V], bool) {
	entry, found := c.items[key]
	if !found {
		var zero Result[V]
		return zero, false
	}

	if entry.expiresAt == 0 {
		return entry.value, true
	}

	now := time.Now().UnixNano()
	if now >= entry.expiresAt {
		delete(c.items, key)
		var zero Result[V]
		return zero, false
	}

	if shouldReset(entry.value, flag) {
		if flag&ResetTimerWith != 0 && duration != 0 {
			entry.duration = duration
		}

		if entry.duration == NoExpiration {
			entry.expiresAt = 0
		} else {
			entry.expiresAt = now + int64(entry.duration)
		}
		c.items[key] = entry
	}
	return entry.value, true
}

func joinFlags(flags []Flag) (flag Flag) {
	for _, current := range flags {
		flag |= current
	}
	return
}

func shouldReset[V any](result Result[V], flag Flag) bool {
	if result.Err != nil {
		return flag&ResetTimerOnErr != 0
	}
	return flag&(ResetTimer|ResetTimerWith) != 0
}

func (c *Cache[K, V]) setUnsafe(key K, value Result[V], duration time.Duration) bool {
	if duration == 0 {
		return false
	}

	if c.limit == 0 {
		delete(c.items, key)
		return false
	}

	if _, found := c.items[key]; !found && c.limit > 0 && len(c.items) >= c.limit {
		c.removeOneUnsafe()
	}

	entry := item[Result[V]]{value: value}
	if duration != NoExpiration {
		entry.duration = duration
		entry.expiresAt = time.Now().UnixNano() + int64(duration)
	}
	c.items[key] = entry
	return true
}

func (c *Cache[K, V]) Remove(key K) bool {
	c.mu.Lock()
	_, found := c.items[key]
	delete(c.items, key)
	c.mu.Unlock()
	return found
}

func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	c.items = make(map[K]item[Result[V]])
	c.mu.Unlock()
}

func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	length := len(c.items)
	c.mu.RUnlock()
	return length
}

func (c *Cache[K, V]) context() context.Context {
	return c.ctx
}

func (c *Cache[K, V]) cleanup(now int64) {
	c.mu.Lock()
	for key, entry := range c.items {
		if entry.expiresAt > 0 && now >= entry.expiresAt {
			delete(c.items, key)
		}
	}
	c.trimUnsafe()
	c.mu.Unlock()
}

func (c *Cache[K, V]) appendItems(items [][]byte, now int64) [][]byte {
	c.mu.RLock()
	for _, entry := range c.items {
		if entry.expiresAt > 0 && now >= entry.expiresAt {
			continue
		}

		value, err := json.Marshal(entry.value.Value)
		if err == nil {
			items = append(items, value)
		}
	}
	c.mu.RUnlock()
	return items
}

func (c *Cache[K, V]) trimUnsafe() {
	for c.limit >= 0 && len(c.items) > c.limit {
		c.removeOneUnsafe()
	}
}

func (c *Cache[K, V]) removeOneUnsafe() {
	for key := range c.items {
		delete(c.items, key)
		return
	}
}

func init() {
	go runCleanup()
}
