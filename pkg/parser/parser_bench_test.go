package parser_test

import (
	"testing"

	"github.com/pyxsoft/php-parser/internal/benchcorpus"
	"github.com/pyxsoft/php-parser/pkg/conf"
	"github.com/pyxsoft/php-parser/pkg/errors"
	"github.com/pyxsoft/php-parser/pkg/parser"
	"github.com/pyxsoft/php-parser/pkg/version"
)

// benchVersion is the PHP version used by the hot path in production.
var benchVersion = &version.Version{Major: 8, Minor: 1}

func parse(tb testing.TB, src []byte) {
	parseWith(tb, src, false)
}

func parseWith(tb testing.TB, src []byte, skipFreeFloating bool) {
	_, err := parser.Parse(src, conf.Config{
		Version:          benchVersion,
		SkipFreeFloating: skipFreeFloating,
		ErrorHandlerFunc: func(e *errors.Error) {
			tb.Fatalf("unexpected parse error: %s", e)
		},
	})
	if err != nil {
		tb.Fatal(err)
	}
}

// TestBenchCorpusParses guarantees the benchmarks measure the success path,
// not error recovery.
func TestBenchCorpusParses(t *testing.T) {
	for _, f := range append(benchcorpus.Synthetic(), benchcorpus.External(t)...) {
		t.Run(f.Name, func(t *testing.T) { parse(t, f.Src) })
	}
}

func BenchmarkParse(b *testing.B) {
	for _, f := range benchcorpus.Synthetic() {
		b.Run(f.Name, func(b *testing.B) {
			b.SetBytes(int64(len(f.Src)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				parse(b, f.Src)
			}
		})
	}
}

// BenchmarkParseSkipFreeFloating measures the AST-only mode used by analyzers
// that do not need whitespace and comments.
func BenchmarkParseSkipFreeFloating(b *testing.B) {
	for _, f := range benchcorpus.Synthetic() {
		b.Run(f.Name, func(b *testing.B) {
			b.SetBytes(int64(len(f.Src)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				parseWith(b, f.Src, true)
			}
		})
	}
}

// BenchmarkParseParallel parses concurrently, as the scanner does; it exposes
// GC pressure and contention that the sequential benchmark hides.
func BenchmarkParseParallel(b *testing.B) {
	for _, f := range benchcorpus.Synthetic() {
		b.Run(f.Name, func(b *testing.B) {
			b.SetBytes(int64(len(f.Src)))
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					parse(b, f.Src)
				}
			})
		})
	}
}

// BenchmarkParseExternal parses the whole external corpus per iteration.
// Set PHPPARSER_BENCH_CORPUS to a directory of PHP files to enable it.
func BenchmarkParseExternal(b *testing.B) {
	files := benchcorpus.External(b)
	if len(files) == 0 {
		b.Skip(benchcorpus.EnvCorpusDir + " not set")
	}
	b.SetBytes(benchcorpus.Size(files))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, f := range files {
			_, _ = parser.Parse(f.Src, conf.Config{Version: benchVersion})
		}
	}
}
