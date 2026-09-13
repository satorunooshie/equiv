// Package hashpool owns reusable maphash scratch values for one hash domain.
package hashpool

import (
	"hash/maphash"
	"sync"
)

type Pool struct {
	seed maphash.Seed
	pool sync.Pool
}

func New(seed maphash.Seed) *Pool {
	p := &Pool{seed: seed}
	p.pool.New = func() any { h := new(maphash.Hash); h.SetSeed(seed); return h }
	return p
}

func Hash[T any](p *Pool, h maphash.Hasher[T], v T) uint64 {
	x := p.pool.Get().(*maphash.Hash)
	x.Reset()
	h.Hash(x, v)
	d := x.Sum64()
	x.Reset()
	p.pool.Put(x)
	return d
}
