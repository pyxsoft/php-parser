package php8_test

import (
	"testing"

	"github.com/pyxsoft/php-parser/internal/benchcorpus"
	"github.com/pyxsoft/php-parser/internal/php8"
	"github.com/pyxsoft/php-parser/pkg/conf"
	"github.com/pyxsoft/php-parser/pkg/version"
)

// BenchmarkLex measures the scanner alone, separating lexing cost from parsing.
func BenchmarkLex(b *testing.B) {
	config := conf.Config{Version: &version.Version{Major: 8, Minor: 1}}

	for _, f := range benchcorpus.Synthetic() {
		b.Run(f.Name, func(b *testing.B) {
			b.SetBytes(int64(len(f.Src)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				lexer := php8.NewLexer(f.Src, config)
				for tkn := lexer.Lex(); tkn.ID != 0; tkn = lexer.Lex() {
				}
			}
		})
	}
}
