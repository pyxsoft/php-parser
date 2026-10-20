package php8

import (
	"bytes"
	"testing"

	"github.com/pyxsoft/php-parser/pkg/conf"
	"github.com/pyxsoft/php-parser/pkg/version"
)

func parseInArena(a *Arena, src []byte) {
	cfg := conf.Config{Version: &version.Version{Major: 8, Minor: 5}}
	NewArenaParser(NewArenaLexer(src, cfg, a), cfg, a).Parse()
}

func TestArenaRetainedLimit(t *testing.T) {
	const limit = 1 << 20
	a := NewArenaWithLimit(limit)

	// a parse that needs more than the limit is released on Reset
	huge := append([]byte("<?php "), bytes.Repeat([]byte("$a = f($b, 1); "), 20000)...)
	parseInArena(a, huge)
	if a.RetainedBytes() <= limit {
		t.Fatalf("test input too small: %d bytes", a.RetainedBytes())
	}
	a.Reset()
	if got := a.RetainedBytes(); got > limit {
		t.Fatalf("retained %d bytes after Reset, limit %d", got, limit)
	}

	// small parses keep their (warm) buffers across Reset
	small := []byte("<?php function f($a) { return $a + 1; }")
	parseInArena(a, small)
	a.Reset()
	warm := a.RetainedBytes()
	for i := 0; i < 10; i++ {
		parseInArena(a, small)
		a.Reset()
	}
	if a.RetainedBytes() != warm {
		t.Fatalf("warm arena changed size: %d -> %d", warm, a.RetainedBytes())
	}
}
