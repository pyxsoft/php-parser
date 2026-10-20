package parser_test

import (
	"bytes"
	"testing"

	"github.com/pyxsoft/php-parser/pkg/ast"
	"github.com/pyxsoft/php-parser/pkg/conf"
	"github.com/pyxsoft/php-parser/pkg/errors"
	"github.com/pyxsoft/php-parser/pkg/parser"
	"github.com/pyxsoft/php-parser/pkg/version"
	"github.com/pyxsoft/php-parser/pkg/visitor/formatter"
	"github.com/pyxsoft/php-parser/pkg/visitor/printer"
)

var php85 = &version.Version{Major: 8, Minor: 5}

// modernSyntax lists PHP 8.2-8.5 constructs the parser must accept.
var modernSyntax = map[string]string{
	"8.2 readonly class":         `<?php final readonly class A { public int $x; }`,
	"8.2 DNF param":              `<?php function f((A&B)|null $x) {}`,
	"8.2 DNF return":             `<?php function f(): (A&B)|(C&D)|false {}`,
	"8.2 DNF property":           `<?php class A { public (X&Y)|null $p; }`,
	"8.2 standalone types":       `<?php function f(null $a): true {} function g(): false {}`,
	"8.3 typed class const":      `<?php class A { const string X = 'a'; final public const int|float Y = 1, Z = 2; }`,
	"8.3 typed const named type": `<?php interface I { const self|Foo\Bar C = null; }`,
	"8.3 dynamic class const":    `<?php $x = A::{$name}; $y = $obj::{'C' . $i};`,
	"8.3 readonly anon class":    `<?php $o = new readonly class(1) extends B {};`,
	"8.4 property hooks":         `<?php class A { public string $n { get => $this->n; set(string $v) { $this->n = strtolower($v); } } }`,
	"8.4 hook default value":     `<?php class A { public int $x = self::D { get => $this->x * 2; } }`,
	"8.4 hook modifiers":         `<?php class A { public int $x { #[A] final get { return 1; } &set => $value; } }`,
	"8.4 abstract hooks":         `<?php interface I { public string $n { get; set; } } abstract class B { abstract public int $x { get; } }`,
	"8.4 empty hook params":      `<?php class A { public int $x { set() {} } }`,
	"8.4 promoted hooks":         `<?php class A { public function __construct(public int $x = 0 { set => $value + 1; }, private $y { get => 2; }) {} }`,
	"8.4 asymmetric visibility":  `<?php class A { public private(set) int $a; protected(set) string $b; public PRIVATE(SET) static int $c; }`,
	"8.4 promoted asymmetric":    `<?php class A { public function __construct(public protected(set) int $x) {} }`,
	"8.4 new without parens":     `<?php new Foo()->bar(); new Foo()::$p; new Foo()['k']; new Foo()::C; new class {}->m(); new Foo()();`,
	"8.4 __PROPERTY__":           `<?php class A { public $x { get => __PROPERTY__; } }`,
	"8.5 pipe":                   `<?php $r = $x |> trim(...) |> strlen(...);`,
	"8.5 void cast":              `<?php (void) foo(); ( void )$a->b();`,
	"8.5 final promoted":         `<?php class A { public function __construct(final public int $x, final $y) {} }`,
	"8.5 attributes on const":    `<?php #[Deprecated] #[A, B(1)] const X = 1, Y = 2;`,
	"8.5 clone with":             `<?php $b = clone($a, ['x' => 1]); $c = clone($a, withProperties: [],); $d = clone($a,); $f = clone(...);`,
	"8.5 clone legacy":           `<?php $b = clone $a; $c = clone($a); $d = clone $a->b;`,
	"8.5 const expr closure":     `<?php class A { const F = static function() {}; const G = strlen(...); }`,
}

func parseStrict(t *testing.T, src string) ast.Vertex {
	t.Helper()

	var errs []*errors.Error
	root, err := parser.Parse([]byte(src), conf.Config{
		Version:          php85,
		ErrorHandlerFunc: func(e *errors.Error) { errs = append(errs, e) },
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range errs {
		t.Errorf("parse error: %s", e)
	}
	if len(errs) > 0 {
		t.FailNow()
	}

	return root
}

func print(n ast.Vertex) string {
	var buf bytes.Buffer
	n.Accept(printer.NewPrinter(&buf))
	return buf.String()
}

func TestModernSyntax(t *testing.T) {
	for name, src := range modernSyntax {
		t.Run(name, func(t *testing.T) {
			root := parseStrict(t, src)

			if got := print(root); got != src {
				t.Fatalf("round trip mismatch\nwant: %s\ngot:  %s", src, got)
			}

			root.Accept(formatter.NewFormatter())
			parseStrict(t, print(root))
		})
	}
}

// stmt returns the n-th top-level statement, unwrapping expression statements.
func stmt(t *testing.T, src string, n int) ast.Vertex {
	t.Helper()

	s := parseStrict(t, src).(*ast.Root).Stmts[n]
	if e, ok := s.(*ast.StmtExpression); ok {
		return e.Expr
	}
	return s
}

func classStmt(t *testing.T, src string, n int) ast.Vertex {
	t.Helper()

	return stmt(t, src, 0).(*ast.StmtClass).Stmts[n]
}

func TestPipePrecedence(t *testing.T) {
	// '.' binds tighter than |>, which binds tighter than ==.
	cmp := stmt(t, `<?php 'a' . 'b' |> f(...) |> g(...) == 2;`, 0).(*ast.ExprBinaryEqual)

	outer := cmp.Left.(*ast.ExprBinaryPipe)
	if _, ok := outer.Right.(*ast.ExprFunctionCall); !ok {
		t.Fatalf("outer pipe right = %T", outer.Right)
	}
	inner := outer.Left.(*ast.ExprBinaryPipe)
	if _, ok := inner.Left.(*ast.ExprBinaryConcat); !ok {
		t.Fatalf("inner pipe left = %T, want concat", inner.Left)
	}

	// arithmetic binds tighter than |>
	pipe := stmt(t, `<?php $a + $b |> f(...);`, 0).(*ast.ExprBinaryPipe)
	if _, ok := pipe.Left.(*ast.ExprBinaryPlus); !ok {
		t.Fatalf("pipe left = %T, want plus", pipe.Left)
	}
}

func TestNewWithoutParentheses(t *testing.T) {
	call := stmt(t, `<?php new Foo(1)->bar();`, 0).(*ast.ExprMethodCall)
	newExpr := call.Var.(*ast.ExprNew)
	if len(newExpr.Args) != 1 {
		t.Fatalf("new args = %d", len(newExpr.Args))
	}

	// without an argument list `new Foo->bar` is still not dereferencable
	if _, ok := stmt(t, `<?php new $foo->bar;`, 0).(*ast.ExprNew); !ok {
		t.Fatal("`new $foo->bar` must stay a new of a property fetch")
	}
}

func TestClone(t *testing.T) {
	legacy := stmt(t, `<?php clone($a);`, 0).(*ast.ExprClone)
	if legacy.Expr == nil || legacy.Args != nil {
		t.Fatal("clone($a) must keep the single-expression form")
	}

	with := stmt(t, `<?php clone($a, ['x' => 1]);`, 0).(*ast.ExprClone)
	if with.Expr != nil || len(with.Args) != 2 || len(with.SeparatorTkns) != 1 {
		t.Fatalf("clone with: expr=%v args=%d seps=%d", with.Expr, len(with.Args), len(with.SeparatorTkns))
	}

	fcc := stmt(t, `<?php clone(...);`, 0).(*ast.ExprClone)
	if fcc.EllipsisTkn == nil || fcc.Args != nil {
		t.Fatal("clone(...) must set EllipsisTkn")
	}
}

func TestPropertyHooks(t *testing.T) {
	list := classStmt(t, `<?php class A { public int $x = 1 { #[A] final &get => 1; set(int $v) { } abstract get; } }`, 0).(*ast.StmtPropertyList)
	if list.SemiColonTkn != nil {
		t.Fatal("hooked property list must not have a semicolon")
	}

	prop := list.Props[0].(*ast.StmtProperty)
	if prop.Expr == nil || len(prop.Hooks) != 3 {
		t.Fatalf("expr=%v hooks=%d", prop.Expr, len(prop.Hooks))
	}

	get := prop.Hooks[0].(*ast.PropertyHook)
	if len(get.AttrGroups) != 1 || len(get.Modifiers) != 1 || get.AmpersandTkn == nil || get.Expr == nil || get.Stmt != nil {
		t.Fatalf("short get hook parsed wrong: %+v", get)
	}
	if string(get.Name.(*ast.Identifier).Value) != "get" {
		t.Fatalf("hook name = %s", get.Name.(*ast.Identifier).Value)
	}

	set := prop.Hooks[1].(*ast.PropertyHook)
	if len(set.Params) != 1 || set.Stmt == nil || set.Expr != nil {
		t.Fatalf("block set hook parsed wrong: %+v", set)
	}

	abstract := prop.Hooks[2].(*ast.PropertyHook)
	if abstract.SemiColonTkn == nil || abstract.Stmt != nil || abstract.Expr != nil {
		t.Fatalf("abstract hook parsed wrong: %+v", abstract)
	}

	promoted := stmt(t, `<?php class A { function __construct(public int $x { get => 1; }) {} }`, 0).(*ast.StmtClass).
		Stmts[0].(*ast.StmtClassMethod).Params[0].(*ast.Parameter)
	if len(promoted.Hooks) != 1 {
		t.Fatalf("promoted hooks = %d", len(promoted.Hooks))
	}
}

func TestDNFTypes(t *testing.T) {
	param := stmt(t, `<?php function f((A&B)|null $x) {}`, 0).(*ast.StmtFunction).Params[0].(*ast.Parameter)
	union := param.Type.(*ast.Union)
	inter := union.Types[0].(*ast.Intersection)
	if inter.OpenParenthesisTkn == nil || len(inter.Types) != 2 {
		t.Fatal("DNF intersection not parsed")
	}
}

func TestTypedClassConstant(t *testing.T) {
	c := classStmt(t, `<?php class A { const string X = 'a'; const array = 1; }`, 0).(*ast.StmtClassConstList)
	if c.Type == nil {
		t.Fatal("typed constant lost its type")
	}

	// a constant named like a type keyword keeps working
	untyped := classStmt(t, `<?php class A { const array = 1; }`, 0).(*ast.StmtClassConstList)
	if untyped.Type != nil {
		t.Fatal("`const array = 1` must not be parsed as a typed constant")
	}
}

func TestDynamicClassConstFetch(t *testing.T) {
	fetch := stmt(t, `<?php A::{$name};`, 0).(*ast.ExprClassConstFetch)
	if _, ok := fetch.Const.(*ast.ExprBrackets); !ok {
		t.Fatalf("const = %T, want ExprBrackets", fetch.Const)
	}
}

func TestAsymmetricVisibility(t *testing.T) {
	list := classStmt(t, `<?php class A { public private(set) int $a; }`, 0).(*ast.StmtPropertyList)
	if got := string(list.Modifiers[1].(*ast.Identifier).Value); got != "private(set)" {
		t.Fatalf("modifier = %q", got)
	}
}

func TestVoidCastAndPropertyMagicConstant(t *testing.T) {
	if _, ok := stmt(t, `<?php (void) f();`, 0).(*ast.ExprCastVoid); !ok {
		t.Fatal("(void) cast not parsed")
	}

	hook := classStmt(t, `<?php class A { public $x { get => __PROPERTY__; } }`, 0).(*ast.StmtPropertyList).
		Props[0].(*ast.StmtProperty).Hooks[0].(*ast.PropertyHook)
	if _, ok := hook.Expr.(*ast.ScalarMagicConstant); !ok {
		t.Fatalf("__PROPERTY__ = %T", hook.Expr)
	}
}

func TestAttributedConstants(t *testing.T) {
	c := stmt(t, `<?php #[A] #[B] const X = 1;`, 0).(*ast.StmtConstList)
	if len(c.AttrGroups) != 2 {
		t.Fatalf("attr groups = %d", len(c.AttrGroups))
	}
}
