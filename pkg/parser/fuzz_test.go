package parser_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pyxsoft/php-parser/internal/benchcorpus"
	"github.com/pyxsoft/php-parser/pkg/ast"
	"github.com/pyxsoft/php-parser/pkg/conf"
	"github.com/pyxsoft/php-parser/pkg/parser"
	"github.com/pyxsoft/php-parser/pkg/version"
)

// FuzzParse checks that no input makes the parser panic: it runs on untrusted
// (often deliberately malformed) code.
func FuzzParse(f *testing.F) {
	for _, src := range modernSyntax {
		f.Add([]byte(src))
	}
	for _, file := range benchcorpus.Synthetic() {
		if len(file.Src) < 64<<10 {
			f.Add(file.Src)
		}
	}
	f.Add([]byte("<?php $a = <<<EOT\n{$x[0]}\n  EOT;\n"))
	f.Add([]byte("<?php class A { public int $x { get => 1; } }"))

	// One Arena per fuzz worker process, reused across inputs: the arena tree
	// must always equal a fresh Parse (differential check for leaked state).
	arena := parser.NewArena()

	f.Fuzz(func(t *testing.T, src []byte) {
		for _, skip := range []bool{false, true} {
			want := parseWithErrs(parser.Parse, src, php85, skip)
			got := parseWithErrs(arena.Parse, src, php85, skip)
			if !reflect.DeepEqual(want, got) {
				t.Fatalf("arena tree differs from Parse (skip=%v)", skip)
			}
			arena.Reset()
		}
	})
}

// TestTrailingLessThan: a '<' as the last byte of inline HTML used to make the
// lexer emit empty T_INLINE_HTML tokens forever (unbounded memory growth).
func TestTrailingLessThan(t *testing.T) {
	for _, src := range []string{"<", "a<", "x\n<", "<<", "<?php ?>\n<", "<?php ?><"} {
		for _, v := range []string{"5.6", "7.4", "8.5"} {
			ver, _ := version.New(v)
			if _, err := parser.Parse([]byte(src), conf.Config{Version: ver}); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// TestAmpersandAtEOF: the '&' token rules looked one byte past the end of the
// token and indexed out of range when '&' ended the input.
func TestAmpersandAtEOF(t *testing.T) {
	spaces := strings.Repeat(" ", 200)
	for _, src := range []string{"<?&0", "<?php &", "<?php function f(&" + spaces + "$a) {}", "<?php $a & " + spaces + "$b;"} {
		parseStrictOrErrors(t, src)
	}

	// more than 100 whitespace bytes between '&' and '$' used to mis-split the token
	root := parseStrict(t, "<?php function f(&"+spaces+"$a) {}")
	if p := root.(*ast.Root).Stmts[0].(*ast.StmtFunction).Params[0].(*ast.Parameter); p.AmpersandTkn == nil || string(p.AmpersandTkn.Value) != "&" {
		t.Fatal("by-ref parameter lost its '&' token")
	}
}

// TestUnbalancedCloseBrace: a top-level '}' used to pop an empty lexer stack.
func TestUnbalancedCloseBrace(t *testing.T) {
	for _, src := range []string{"<?}", "<?php } } }", "<?php \"{$a}}\"; }"} {
		parseStrictOrErrors(t, src)
	}
}

// TestLookaheadAtEOF: lexer lookahead helpers indexed past the end of input.
func TestLookaheadAtEOF(t *testing.T) {
	for _, src := range []string{
		"<?php $a;\r", "<?php \"$", "<?php \"{", "<?php \"\\", "<?php `$", "<?php C\"000$",
		"<?php <<<EOT\n", "<?php <<<EOT\n  ", "<?php <<<EOT\nx\n\t ", "<?php <<<'EOT'\n", "<?php #", "<?php //",
	} {
		parseStrictOrErrors(t, src)
	}
}

// parseStrictOrErrors parses src, tolerating syntax errors but not panics.
func parseStrictOrErrors(t *testing.T, src string) {
	t.Helper()
	if _, err := parser.Parse([]byte(src), conf.Config{Version: php85}); err != nil {
		t.Fatal(err)
	}
}

// TestEmptyHeredoc: the closing label of an empty heredoc lost its first byte
// (and indexed past the input when the label ended the file).
func TestEmptyHeredoc(t *testing.T) {
	for _, src := range []string{"<?php $x = <<<EOT\nEOT;\n", "<?php $x = <<<EOT\n    EOT;\n", "<?php $x = <<<'EOT'\nEOT;\n"} {
		root := parseStrict(t, src)
		if got := print(root); got != src {
			t.Fatalf("round trip mismatch\nwant: %q\ngot:  %q", src, got)
		}

		heredoc := root.(*ast.Root).Stmts[0].(*ast.StmtExpression).Expr.(*ast.ExprAssign).Expr.(*ast.ScalarHeredoc)
		if string(heredoc.CloseHeredocTkn.Value) != "EOT" {
			t.Fatalf("closing label = %q", heredoc.CloseHeredocTkn.Value)
		}
	}

	parseStrictOrErrors(t, "<?<<<A\nA ")
}

// TestHeredocVariableVariable: "$$name" at the start of a heredoc segment made
// the lexer bounce between the heredoc and string_var states forever.
func TestHeredocVariableVariable(t *testing.T) {
	for _, src := range []string{
		"<?php echo <<<EOT\n$$amount\nEOT;\n",
		"<?php echo <<<EOT\nTotal: {$a}$$b\nEOT;\n",
		"<?php echo <<<EOT\n$\nEOT;\n",
		"<?<<<A00\n$$x[}\n  [0]}\n  EEOT\n",
	} {
		done := make(chan struct{})
		go func() {
			defer close(done)
			parseStrictOrErrors(t, src)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("parser hangs on %q", src)
		}
	}

	parts := parseStrict(t, "<?php echo <<<EOT\n$$amount\nEOT;\n").(*ast.Root).Stmts[0].(*ast.StmtEcho).Exprs[0].(*ast.ScalarHeredoc).Parts
	if len(parts) != 3 {
		t.Fatalf("parts = %d, want literal '$', $amount, newline", len(parts))
	}
	if _, ok := parts[1].(*ast.ExprVariable); !ok {
		t.Fatalf("parts[1] = %T, want variable", parts[1])
	}
}
