package leerlauf

import (
	"google.golang.org/appengine/memcache"
	"context"
	"time"
	"strconv"
	"google.golang.org/appengine/log"
)

type limit struct {
	description string
	context context.Context
	max uint64
}

// TODO key must not be longer than 250 bytes in total according to docs
func NewLimit(description string, max uint64) *limit {
	return &limit{description: description, max: max}
}

func (l limit) Limited(ctx context.Context, id string) bool {
	l.context = ctx

	if l.isMitigated(id) {
		return true
	}

	now := time.Now()
	before := now.Add(-1 * time.Minute)

	nowCounter := l.getCounter(id, now.Minute())
	beforeCounter := l.getCounter(id, before.Minute())

	sixty := uint64(60)
	rate := uint64(beforeCounter * ((sixty - uint64(now.Second())) / sixty) + nowCounter)

	log.Infof(ctx, "BeforeCounter: %v, nowSecond: %v, nowCounter: %v, Rate: %v", beforeCounter, now.Second(), nowCounter, rate)

	if rate > l.max {
		l.mitigate(id)
		return true
	}

	l.setCounter(id, now.Minute())
	return false
}

func (l limit) mitigate(id string) {
	memcache.Set(l.context, &memcache.Item{
		Key: l.createKey(id) + ":mitigated",
		Value: []byte{1},
		Expiration: time.Minute,
	})
}

func (l limit) isMitigated(id string) bool {
	key := l.createKey(id) + ":mitigated"
	_, err := memcache.Get(l.context, key)
	return err != memcache.ErrCacheMiss
}

func (l limit) createKey(id string) string {
	return l.description + id
}

func (l limit) getCounter(id string, minute int) uint64 {
	key := l.createKey(id) + ":" + strconv.Itoa(minute)
	res, err := memcache.Increment(l.context, key, int64(0), uint64(1))
	if err == memcache.ErrCacheMiss {
		return 0
	}

	if err != nil && err != memcache.ErrCacheMiss {
		log.Errorf(l.context, "Error on reading Counter: %v", err)
	}

	return res
}

func (l limit) setCounter(id string, minute int) {
	key := l.createKey(id) + ":" + strconv.Itoa(minute)
	log.Infof(l.context, "Key: %v", key)
	my, err := memcache.Increment(l.context, key, int64(1), uint64(1))
	if err != nil {
		log.Errorf(l.context, "Error on incrementing Counter: %v", err)
	}
	log.Infof(l.context, "New value: %v", my)
}
