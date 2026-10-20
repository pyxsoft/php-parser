package position

const DefaultBlockSize = 1024

// minBlockSize is the smallest first block NewPoolFor allocates.
const minBlockSize = 32

// Pool hands out Positions from preallocated blocks to avoid one heap
// allocation per Position.
type Pool struct {
	block []Position
	off   int
	max   int
}

// NewPool returns a pool that allocates blocks of blockSize Positions.
func NewPool(blockSize int) *Pool {
	return &Pool{
		block: make([]Position, blockSize),
		max:   blockSize,
	}
}

// NewPoolFor returns a pool whose first block holds hint Positions (clamped to
// [minBlockSize, maxBlockSize]); later blocks double in size up to
// maxBlockSize. It avoids paying for a full block on small inputs.
func NewPoolFor(hint, maxBlockSize int) *Pool {
	size := min(max(hint, minBlockSize), maxBlockSize)

	return &Pool{
		block: make([]Position, size),
		max:   maxBlockSize,
	}
}

func (p *Pool) Get() *Position {
	if len(p.block) == 0 {
		return nil
	}

	if len(p.block) == p.off {
		p.block = make([]Position, min(len(p.block)*2, max(p.max, len(p.block))))
		p.off = 0
	}

	p.off++

	return &p.block[p.off-1]
}
