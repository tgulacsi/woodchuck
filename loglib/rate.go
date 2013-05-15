package loglib

// implements a simple time-evicted rate-limiting map

import (
	"hash/fnv"
	"sync"
	"time"
)

type RateLimiter interface {
	Put(time.Duration, string) bool
}

type nextMap struct {
	m map[uint64]time.Time
	sync.RWMutex
}

func NewRateLimiter(eviction time.Duration) *nextMap {
	rl := &nextMap{m: make(map[uint64]time.Time, 16)}
	if eviction > 0 {
		go func() {
			for n := range time.Tick(time.Hour) {
				rl.Lock()
				for k, v := range rl.m {
					if v.Before(n) {
						delete(rl.m, k)
					}
				}
				rl.Unlock()
			}
		}()
	}
	return rl
}

func getHash(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

// checks whether s has been put in the map before, and whether the eviction time of it has been over.
// if s is not in them map or the time is over, than register the string with Now() + n eviction time and return true
// else return false
func (nm *nextMap) Put(n time.Duration, s string) bool {
	h := getHash(s)
	nm.RLock()
	if t, ok := nm.m[h]; ok && t.After(time.Now()) {
		nm.RUnlock()
		return false
	}
	nm.RUnlock()
	nm.Lock()
	nm.m[h] = time.Now().Add(n)
	nm.Unlock()
	return true
}
