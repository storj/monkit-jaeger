// Copyright (C) 2020 Storj Labs, Inc.
// Copyright (C) 2014 Space Monkey, Inc.
// See LICENSE for copying information.

package jaeger

import (
	crypto_rand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"sync"
)

type locker struct {
	l sync.Mutex
	s rand.Source
}

func (l *locker) Int63() (rv int64) {
	l.l.Lock()
	rv = l.s.Int63()
	l.l.Unlock()
	return rv
}

func (l *locker) Seed(seed int64) {
	l.l.Lock()
	l.s.Seed(seed)
	l.l.Unlock()
}

func seed() int64 {
	var seed [8]byte
	_, err := crypto_rand.Read(seed[:])
	if err != nil {
		panic(err)
	}
	return int64(binary.BigEndian.Uint64(seed[:]))
}

var rng = rand.New(&locker{s: rand.NewSource(seed())})
