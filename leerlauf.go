package leerlauf

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/appengine/memcache"
	"strconv"
	"time"
)

type limit struct {
	description string
	context     context.Context
	max         int
}

// ErrMitigated is returned on a successful mitigation
var ErrMitigated = errors.New("Access mitigated")

const mitigated = ":mitigated"
const maxMemcacheKey = 250

// NewLimit receives a unique description and the maximum hits
// per minute before mitigation starts for a given id.
func NewLimit(description string, maxHits int) (*limit, error) {
	const maxBytes = 248
	if len(description) > maxBytes {
		return nil, fmt.Errorf("Max len for description is %v bytes. Got %v bytes.",	maxBytes, len(description))
	}
	return &limit{description: description, max: maxHits}, nil
}

// Limited returns nil if given id did not exceed the limit.
// For mitigations it returns ErrMitigated, else errors from
// datastore that might bubble up.
func (l limit) Limited(ctx context.Context, id string) error {
	l.context = ctx

	if len(id)+len(l.description)+1 > maxMemcacheKey {
		return errors.New(
			"Sum of given id plus description is too long")
	}

	if len(id)+len(mitigated) > maxMemcacheKey {
		return errors.New(
			"Given id is too long")
	}

	m, err := l.isMitigated(id)
	if err != nil {
		return err
	}

	if m {
		return ErrMitigated
	}

	now := time.Now()
	before := now.Add(-1 * time.Minute)

	nowCounter, err := l.getCounter(id, now.Minute())
	if err != nil {
		return err
	}

	beforeCounter, err := l.getCounter(id, before.Minute())
	if err != nil {
		return err
	}

	rate := beforeCounter*((60-now.Second())/60.0) + nowCounter

	if rate > l.max {
		err = l.mitigate(id)
		if err != nil {
			return err
		}
		return ErrMitigated
	}

	err = l.setCounter(id, now.Minute())
	if err != nil {
		return err
	}
	return nil
}

func (l limit) mitigate(id string) error {
	return memcache.Set(l.context, &memcache.Item{
		Key:        l.createKey(id) + mitigated,
		Value:      []byte{1},
		Expiration: time.Minute,
	})
}

func (l limit) isMitigated(id string) (bool, error) {
	key := l.createKey(id) + mitigated
	_, err := memcache.Get(l.context, key)
	if err == memcache.ErrCacheMiss {
		return false, nil
	}

	if err == nil {
		return true, nil
	}

	return true, err
}

func (l limit) createKey(id string) string {
	return l.description + id
}

func (l limit) getCounter(id string, minute int) (int, error) {
	key := l.createKey(id) + ":" + strconv.Itoa(minute)
	res, err := memcache.Increment(l.context, key, int64(0), uint64(1))
	if err == memcache.ErrCacheMiss {
		return 0, nil
	}

	if err != nil && err != memcache.ErrCacheMiss {
		return 0, err
	}

	return int(res), nil
}

func (l limit) setCounter(id string, minute int) error {
	key := l.createKey(id) + ":" + strconv.Itoa(minute)
	_, err := memcache.Increment(l.context, key, int64(1), uint64(1))
	return err
}
