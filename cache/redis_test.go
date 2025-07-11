// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package cache

import (
	"net"
	"testing"
	"time"

	"github.com/enigma-id/go/env"
)

// These tests require redis server running on localhost:6379 (the default)
var newRedisCache = func(t *testing.T, defaultExpiration time.Duration) Cache {
	redisTestServer := env.GetString("REDIS_HOST", "localhost:6379")
	c, err := net.Dial("tcp", redisTestServer)
	if err == nil {
		if _, err = c.Write([]byte("flush_all\r\n")); err != nil {
			t.Errorf("Write failed: %s", err)
		}
		_ = c.Close()

		redisCache := NewRedisCache()
		if err = redisCache.Flush(); err != nil {
			t.Errorf("Flush failed: %s", err)
		}
		return redisCache
	}
	t.Errorf("couldn't connect to redis on %s", redisTestServer)
	t.FailNow()
	panic("")
}

func TestRedisCache_TypicalGetSet(t *testing.T) {
	typicalGetSet(t, newRedisCache)
}

func TestRedisCache_Expiration(t *testing.T) {
	expiration(t, newRedisCache)
}

func TestRedisCache_EmptyCache(t *testing.T) {
	emptyCache(t, newRedisCache)
}

func TestRedisCache_Replace(t *testing.T) {
	testReplace(t, newRedisCache)
}

func TestRedisCache_Add(t *testing.T) {
	testAdd(t, newRedisCache)
}

func TestRedisCache_GetMulti(t *testing.T) {
	testGetMulti(t, newRedisCache)
}
