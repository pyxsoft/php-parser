package parser

import (
	"github.com/pyxsoft/php-parser/internal/php8"
	"github.com/pyxsoft/php-parser/pkg/ast"
	"github.com/pyxsoft/php-parser/pkg/conf"
)

// Arena parses many files in a row reusing the same memory. Tokens,
// positions and AST nodes of every tree it returns live in the Arena and stay
// valid only until Reset:
//
//	arena := parser.NewArena() // one per goroutine
//	for _, src := range files {
//		root, err := arena.Parse(src, cfg)
//		// ... use root ...
//		arena.Reset() // root and everything reachable from it is now invalid
//	}
//
// Copy whatever must outlive the tree (e.g. a *position.Position) before
// calling Reset. After warm-up a parse allocates far less than Parse and puts
// almost no pressure on the garbage collector. Only PHP 8 sources use the
// Arena; PHP 5 and 7 fall back to Parse.
//
// # Concurrency
//
// An Arena is not safe for concurrent use and has no internal locking, on
// purpose: give each goroutine its own Arena. A sync.Pool works well for
// handlers that run on arbitrary goroutines:
//
//	var arenas = sync.Pool{New: func() any { return parser.NewArena() }}
//
//	arena := arenas.Get().(*parser.Arena)
//	root, err := arena.Parse(src, cfg)
//	// ... use root; nothing that points into it may escape ...
//	arena.Reset()
//	arenas.Put(arena)
//
// Reset before Put, never after: once an Arena is back in the pool another
// goroutine may reuse its memory. If parsing or the analysis of the tree
// panics, drop the Arena instead of putting it back. A sync.Pool may discard
// idle Arenas at any GC cycle; use a fixed set (e.g. one per worker, or a
// buffered channel) when retained memory must be bounded and predictable.
//
// # Memory
//
// The buffers an Arena needs grow with the input: about 20-50x the source
// size for typical code and up to ~100x for minified code (measured). Reset
// keeps them for reuse only while they stay within the Arena's limit
// (DefaultMaxRetained, 16 MiB, unless set with NewArenaWithLimit); a larger
// parse still uses the Arena, but Reset then releases its buffers. So an idle
// Arena never holds more than its limit: budget limit x live Arenas.
//
// Choosing a limit: an Arena stays warm for files up to about limit/50. In
// real-world corpora (~66k PHP files from WordPress, plugins, frameworks) the
// median file is 2 KB, 99% are under 100 KB and 99.9% under 300 KB. Hence:
//
//   - 4 MiB: warm for files up to ~100 KB (99% of files). For many workers or
//     memory-constrained hosts: 16 workers retain at most 64 MiB.
//   - 16 MiB (DefaultMaxRetained): warm up to ~300 KB (99.9% of files). The
//     balanced choice: 8 workers retain at most 128 MiB.
//   - More than 32 MiB rarely pays off: the remaining files are too few for
//     buffer reuse to matter, and they still parse through the Arena.
type Arena struct {
	a *php8.Arena
}

// DefaultMaxRetained is the memory an Arena from NewArena keeps across Reset.
const DefaultMaxRetained = php8.DefaultMaxRetained

func NewArena() *Arena {
	return &Arena{a: php8.NewArena()}
}

// NewArenaWithLimit returns an Arena that keeps at most maxRetained bytes of
// buffers across Reset.
func NewArenaWithLimit(maxRetained int) *Arena {
	return &Arena{a: php8.NewArenaWithLimit(maxRetained)}
}

// RetainedBytes is the memory currently held by the Arena's buffers.
func (a *Arena) RetainedBytes() int {
	return a.a.RetainedBytes()
}

// Parse is like the package-level Parse but allocates from the Arena.
func (a *Arena) Parse(src []byte, config conf.Config) (ast.Vertex, error) {
	if config.Version == nil || !config.Version.InRange(php8RangeStart, php8RangeEnd) {
		return Parse(src, config)
	}

	lexer := php8.NewArenaLexer(src, config, a.a)
	p := php8.NewArenaParser(lexer, config, a.a)
	p.Parse()

	return p.GetRootNode(), nil
}

// Reset invalidates all trees returned since the previous Reset and makes
// their memory available to the next Parse.
func (a *Arena) Reset() {
	a.a.Reset()
}
