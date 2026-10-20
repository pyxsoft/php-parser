package parser_test

import (
	"testing"

	"github.com/pyxsoft/php-parser/pkg/ast"
	"github.com/pyxsoft/php-parser/pkg/parser"
	"github.com/pyxsoft/php-parser/pkg/version"
)

// TestReadonlyFunctionName: readonly is a modifier but still a valid function
// name, declared and called (WordPress wp-includes/php-compat/readonly.php),
// without breaking readonly classes and properties.
func TestReadonlyFunctionName(t *testing.T) {
	valid := map[string][]string{
		"8.1": {
			"<?php function readonly($a) { return $a; }",
			"<?php function &readonly() {}",
			"<?php #[A] function readonly() {}",
			"<?php echo readonly(1), readonly (2), \\readonly(3);",
			"<?php if (true) { readonly($x); }",
			"<?php class C { static function readonly() {} function f() { $this->readonly(); C::readonly(); } }",
			"<?php class A { readonly int $a; public function __construct(readonly int $b, private readonly ?A $c) {} }",
		},
		"8.2": {
			"<?php readonly class B {}",
			"<?php #[A] final readonly class B {}",
			"<?php class A { public readonly (A&B)|null $p; readonly (A&B)|C $q; }",
		},
		"8.3": {
			"<?php $o = new readonly class {};",
		},
	}

	for _, v := range []string{"8.1", "8.5"} {
		ver, _ := version.New(v)
		for since, srcs := range valid {
			if since > v {
				continue
			}
			for _, src := range srcs {
				if r := parseWithErrs(parser.Parse, []byte(src), ver, false); len(r.errs) > 0 {
					t.Errorf("%s %q: %v", v, src, r.errs)
				}
			}
		}
	}

	r := parseWithErrs(parser.Parse, []byte("<?php function readonly() {} readonly(1);"), php85, false)
	stmts := r.root.(*ast.Root).Stmts
	if fn, ok := stmts[0].(*ast.StmtFunction); !ok || string(fn.Name.(*ast.Identifier).Value) != "readonly" {
		t.Errorf("declaration: got %#v", stmts[0])
	}
	call, ok := stmts[1].(*ast.StmtExpression).Expr.(*ast.ExprFunctionCall)
	if !ok || len(call.Args) != 1 {
		t.Fatalf("call: got %#v", stmts[1])
	}
	if name, ok := call.Function.(*ast.Name); !ok || string(name.Parts[0].(*ast.NamePart).Value) != "readonly" {
		t.Errorf("call name: got %#v", call.Function)
	}
}
