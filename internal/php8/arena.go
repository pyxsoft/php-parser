package php8

import (
	"unsafe"

	"github.com/pyxsoft/php-parser/pkg/position"
	"github.com/pyxsoft/php-parser/pkg/token"
)

// DefaultMaxRetained is the default bound on the memory an Arena keeps
// across Reset. A typical 200 KB PHP file needs about 4 MB.
const DefaultMaxRetained = 16 << 20

// Arena holds the buffers that parsing allocates (tokens, positions, AST
// nodes, the line table) so that many files can be parsed one after another
// without allocating them again. It is not safe for concurrent use.
type Arena struct {
	tokens  *token.Pool
	lexPos  *position.Pool
	nodePos *position.Pool
	nodes   nodeAlloc

	newLines []int
	stack    []int
	lexer    *Lexer // last lexer, to take back its grown buffers

	maxRetained int
}

// NewArena returns an Arena that keeps at most DefaultMaxRetained bytes
// across Reset.
func NewArena() *Arena {
	return NewArenaWithLimit(DefaultMaxRetained)
}

// NewArenaWithLimit returns an Arena that keeps at most maxRetained bytes of
// buffers across Reset; if a parse grew it beyond that, Reset releases them.
func NewArenaWithLimit(maxRetained int) *Arena {
	return &Arena{
		tokens:      token.NewPool(token.DefaultBlockSize),
		lexPos:      position.NewPool(position.DefaultBlockSize),
		nodePos:     position.NewPool(position.DefaultBlockSize),
		maxRetained: maxRetained,
	}
}

// Reset invalidates every tree, token and position handed out since the
// previous Reset and makes their memory available for the next parse. Used
// slots are zeroed, so stale references fail fast instead of reading another
// file's data, and the previous sources are no longer kept alive. Buffers
// larger than the Arena's limit are released instead of kept.
func (a *Arena) Reset() {
	if a.lexer != nil {
		a.newLines = a.lexer.newLines.data[:0]
		a.stack = a.lexer.stack[:0]
		a.lexer = nil
	}

	if a.RetainedBytes() > a.maxRetained {
		*a = *NewArenaWithLimit(a.maxRetained)
		return
	}

	a.tokens.Reset()
	a.lexPos.Reset()
	a.nodePos.Reset()
	a.nodes.reset()
}

// RetainedBytes is the memory held by the Arena's buffers.
func (a *Arena) RetainedBytes() int {
	const (
		tokenSize    = int(unsafe.Sizeof(token.Token{}))
		positionSize = int(unsafe.Sizeof(position.Position{}))
		intSize      = int(unsafe.Sizeof(int(0)))
	)

	return a.tokens.Cap()*tokenSize +
		(a.lexPos.Cap()+a.nodePos.Cap())*positionSize +
		a.nodes.capBytes() +
		(cap(a.newLines)+cap(a.stack))*intSize
}
