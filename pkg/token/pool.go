package token

const DefaultBlockSize = 1024

// minBlockSize is the smallest first block NewPoolFor allocates.
const minBlockSize = 32

// Pool hands out Tokens from preallocated blocks to avoid one heap
// allocation per Token.
type Pool struct {
	block []Token
	off   int
	max   int
}

// NewPool returns a pool that allocates blocks of blockSize Tokens.
func NewPool(blockSize int) *Pool {
	return &Pool{
		block: make([]Token, blockSize),
		max:   blockSize,
	}
}

// NewPoolFor returns a pool whose first block holds hint Tokens (clamped to
// [minBlockSize, maxBlockSize]); later blocks double in size up to
// maxBlockSize. It avoids paying for a full block on small inputs.
func NewPoolFor(hint, maxBlockSize int) *Pool {
	size := min(max(hint, minBlockSize), maxBlockSize)

	return &Pool{
		block: make([]Token, size),
		max:   maxBlockSize,
	}
}

func (p *Pool) Get() *Token {
	if len(p.block) == 0 {
		return nil
	}

	if len(p.block) == p.off {
		p.block = make([]Token, min(len(p.block)*2, max(p.max, len(p.block))))
		p.off = 0
	}

	p.off++

	return &p.block[p.off-1]
}
