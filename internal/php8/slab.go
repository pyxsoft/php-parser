package php8

import "unsafe"

//go:generate go run ./genalloc

// slab hands out T values from blocks that start small and double up to
// maxSlabBlock. reset zeroes and rewinds it so a reusable Arena can parse the
// next file without allocating.
type slab[T any] struct {
	blocks [][]T
	cur    int
	off    int
}

const (
	minSlabBlock = 8
	maxSlabBlock = 512
)

func newNode[T any](s *slab[T], v T) *T {
	switch {
	case len(s.blocks) == 0:
		s.blocks = append(s.blocks, make([]T, minSlabBlock))
	case s.off == len(s.blocks[s.cur]):
		s.cur++
		s.off = 0
		if s.cur == len(s.blocks) {
			s.blocks = append(s.blocks, make([]T, min(len(s.blocks[s.cur-1])*2, maxSlabBlock)))
		}
	}

	p := &s.blocks[s.cur][s.off]
	s.off++
	*p = v

	return p
}

// capBytes is the memory held by the slab's blocks.
func (s *slab[T]) capBytes() int {
	var zero T
	n := 0
	for _, b := range s.blocks {
		n += len(b)
	}

	return n * int(unsafe.Sizeof(zero))
}

func (s *slab[T]) reset() {
	if len(s.blocks) == 0 {
		return
	}

	for i := 0; i < s.cur; i++ {
		clear(s.blocks[i])
	}
	clear(s.blocks[s.cur][:s.off])

	s.cur = 0
	s.off = 0
}
