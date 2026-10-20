package parser_test

import (
	"reflect"
	"testing"

	"github.com/pyxsoft/php-parser/internal/benchcorpus"
	"github.com/pyxsoft/php-parser/pkg/ast"
	"github.com/pyxsoft/php-parser/pkg/conf"
	"github.com/pyxsoft/php-parser/pkg/errors"
	"github.com/pyxsoft/php-parser/pkg/parser"
	"github.com/pyxsoft/php-parser/pkg/version"
)

type parseResult struct {
	root ast.Vertex
	errs []string
}

func parseWithErrs(parse func([]byte, conf.Config) (ast.Vertex, error), src []byte, v *version.Version, skip bool) parseResult {
	var r parseResult
	root, err := parse(src, conf.Config{
		Version:          v,
		SkipFreeFloating: skip,
		ErrorHandlerFunc: func(e *errors.Error) { r.errs = append(r.errs, e.String()) },
	})
	if err != nil {
		r.errs = append(r.errs, err.Error())
	}
	r.root = root
	return r
}

func arenaSources(t *testing.T) [][]byte {
	var srcs [][]byte
	for _, f := range benchcorpus.Synthetic() {
		srcs = append(srcs, f.Src)
	}
	for _, src := range modernSyntax {
		srcs = append(srcs, []byte(src))
	}
	for _, f := range benchcorpus.External(t) {
		srcs = append(srcs, f.Src)
	}
	return append(srcs,
		[]byte("<?php $x = <<<EOT\n  {$a['b']} $c->d\n  EOT;\n"),
		[]byte("<?php syntax error here (("),
		[]byte("<html><?= $a ?></html>"),
	)
}

// TestArenaMatchesParse parses every source with a single reused Arena and
// checks each tree is identical to a fresh Parse: state leaking from one file
// into the next would show up here.
func TestArenaMatchesParse(t *testing.T) {
	srcs := arenaSources(t)
	arena := parser.NewArena()

	for round := 0; round < 2; round++ {
		for _, skip := range []bool{false, true} {
			for i, src := range srcs {
				want := parseWithErrs(parser.Parse, src, php85, skip)
				got := parseWithErrs(arena.Parse, src, php85, skip)

				if !reflect.DeepEqual(want, got) {
					t.Fatalf("round %d skip=%v source #%d: arena tree differs from Parse", round, skip, i)
				}
				arena.Reset()
			}
		}
	}
}

// TestArenaResetInvalidates checks that Reset zeroes the previous tree so a
// stale reference fails fast and the old source is not kept alive.
func TestArenaResetInvalidates(t *testing.T) {
	arena := parser.NewArena()
	root, _ := arena.Parse([]byte(`<?php echo "hi";`), conf.Config{Version: php85})

	echo := root.(*ast.Root).Stmts[0].(*ast.StmtEcho)
	tkn := echo.EchoTkn
	if string(tkn.Value) != "echo" || echo.Position == nil {
		t.Fatal("unexpected tree")
	}

	arena.Reset()

	if tkn.Value != nil || tkn.Position != nil || echo.Position != nil || echo.Exprs != nil {
		t.Fatal("Reset left stale data in the previous tree")
	}
}

// TestArenaOlderVersions: PHP 5/7 parse through the regular path.
func TestArenaOlderVersions(t *testing.T) {
	arena := parser.NewArena()
	for _, v := range []string{"5.6", "7.4"} {
		ver, _ := version.New(v)
		src := []byte(`<?php function f($a) { return $a + 1; }`)
		if !reflect.DeepEqual(parseWithErrs(parser.Parse, src, ver, false), parseWithErrs(arena.Parse, src, ver, false)) {
			t.Fatalf("%s: arena differs", v)
		}
		arena.Reset()
	}
}
