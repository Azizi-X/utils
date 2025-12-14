<<<<<<< HEAD
package utils

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type item struct {
	mu   sync.Mutex
	time time.Time
	ref  atomic.Int32
}

type Limiter struct {
	values   map[string]*item
	cooldown time.Duration
	mu       sync.Mutex
	ctx      context.Context
}

func NewLimiter(cooldown time.Duration, ctx context.Context) *Limiter {
	limiter := &Limiter{
		values:   make(map[string]*item),
		cooldown: cooldown,
		ctx:      ctx,
	}

	go limiter.loop()

	return limiter
}

func (l *Limiter) loop() {
	for l.ctx.Err() == nil {
		time.Sleep(1 * time.Second)
		l.mu.Lock()
		for key, value := range l.values {
			if value.ref.Load() <= 0 && time.Since(value.time) > l.cooldown {
				delete(l.values, key)
			}
		}
		l.mu.Unlock()
	}
}

func (l *Limiter) LimitKey(key string) {
	l.mu.Lock()
	value, ok := l.values[key]

	if !ok {
		l.values[key] = &item{
			time: time.Now(),
		}
		value = l.values[key]
	}

	value.ref.Add(1)

	l.mu.Unlock()

	defer value.ref.Add(-1)

	if !ok {
		return
	}

	value.mu.Lock()
	defer value.mu.Unlock()

	remaining := l.cooldown - time.Since(value.time)
	if remaining > 0 {
		time.Sleep(remaining)
	}

	value.time = time.Now()
}
=======
package utils

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type item struct {
	mu   sync.Mutex
	time time.Time
	ref  atomic.Int32
}

type Limiter struct {
	values   map[string]*item
	cooldown time.Duration
	mu       sync.Mutex
	ctx      context.Context
}

func NewLimiter(cooldown time.Duration, ctx context.Context) *Limiter {
	limiter := &Limiter{
		values:   make(map[string]*item),
		cooldown: cooldown,
		ctx:      ctx,
	}

	go limiter.loop()

	return limiter
}

func (l *Limiter) loop() {
	for l.ctx.Err() == nil {
		time.Sleep(1 * time.Second)
		l.mu.Lock()
		for key, value := range l.values {
			if value.ref.Load() <= 0 && time.Since(value.time) > l.cooldown {
				delete(l.values, key)
			}
		}
		l.mu.Unlock()
	}
}

func (l *Limiter) LimitKey(key string) {
	l.mu.Lock()
	value, ok := l.values[key]

	if !ok {
		l.values[key] = &item{
			time: time.Now(),
		}
		value = l.values[key]
	}

	value.ref.Add(1)

	l.mu.Unlock()

	defer value.ref.Add(-1)

	if !ok {
		return
	}

	value.mu.Lock()
	defer value.mu.Unlock()

	remaining := l.cooldown - time.Since(value.time)
	if remaining > 0 {
		time.Sleep(remaining)
	}

	value.time = time.Now()
}
>>>>>>> 2330299ae49ffbbd1dbfff4a195c0169abca5303
