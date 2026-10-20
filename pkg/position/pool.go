package position

const DefaultBlockSize = 1024

// minBlockSize is the smallest first block NewPoolFor allocates.
const minBlockSize = 32

// Pool hands out Positions from preallocated blocks to avoid one heap
// allocation per Position. Reset rewinds it so the blocks can be reused.
type Pool struct {
	blocks [][]Position
	cur    int // block being filled
	off    int // next free slot in blocks[cur]
	max    int
}

// NewPool returns a pool that allocates blocks of blockSize Positions.
func NewPool(blockSize int) *Pool {
	return &Pool{
		blocks: [][]Position{make([]Position, blockSize)},
		max:    blockSize,
	}
}

// NewPoolFor returns a pool whose first block holds hint Positions (clamped to
// [minBlockSize, maxBlockSize]); later blocks double in size up to
// maxBlockSize. It avoids paying for a full block on small inputs.
func NewPoolFor(hint, maxBlockSize int) *Pool {
	size := min(max(hint, minBlockSize), maxBlockSize)

	return &Pool{
		blocks: [][]Position{make([]Position, size)},
		max:    maxBlockSize,
	}
}

func (p *Pool) Get() *Position {
	if len(p.blocks) == 0 || len(p.blocks[0]) == 0 {
		return nil
	}

	if p.off == len(p.blocks[p.cur]) {
		p.cur++
		p.off = 0

		if p.cur == len(p.blocks) {
			prev := len(p.blocks[p.cur-1])
			p.blocks = append(p.blocks, make([]Position, min(prev*2, max(p.max, prev))))
		}
	}

	p.off++

	return &p.blocks[p.cur][p.off-1]
}

// Reset zeroes every Position handed out since the last Reset and rewinds the
// pool, keeping its blocks. Pointers obtained before Reset must not be used
// afterwards.
func (p *Pool) Reset() {
	if len(p.blocks) == 0 {
		return
	}

	for i := 0; i < p.cur; i++ {
		clear(p.blocks[i])
	}
	clear(p.blocks[p.cur][:p.off])

	p.cur = 0
	p.off = 0
}

// Cap returns the number of Positions the pool can hand out without allocating.
func (p *Pool) Cap() int {
	n := 0
	for _, b := range p.blocks {
		n += len(b)
	}

	return n
}
