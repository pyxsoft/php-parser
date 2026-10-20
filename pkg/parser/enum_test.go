package parser_test

import (
	"testing"

	"github.com/pyxsoft/php-parser/pkg/ast"
	"github.com/pyxsoft/php-parser/pkg/parser"
	"github.com/pyxsoft/php-parser/pkg/version"
)

// TestEnumAsName: "enum" is a keyword only when a name other than extends or
// implements follows it, so classes named Enum (myclabs/php-enum, Laravel,
// Doctrine) still parse with the 8.x grammar. Comments between enum and the
// name are accepted by PHP since 8.3, and here by every 8.x version.
func TestEnumAsName(t *testing.T) {
	names := []string{
		"<?php class Enum {}",
		"<?php abstract class Enum extends Base implements \\JsonSerializable {}",
		"<?php class Enum /* c */ extends Base {}",
		"<?php class Enum\n\timplements I {}",
		"<?php $a = new Enum(1);",
		"<?php $a = Enum::FOO;",
		"<?php if ($a instanceof Enum) {}",
		"<?php function f(Enum $e): Enum { return $e; }",
		"<?php use Foo\\Enum;",
		"<?php interface Enum {} trait ENUM {}",
		"<?php class A { function enum() {} const enum = 1; } A::enum(); $a->enum;",
		"<?php $a = enum ?>",
	}
	decls := []string{
		"<?php enum A {}",
		"<?php ENUM A: string implements B { case X = 'x'; }",
		"<?php enum /* c */ A {}",
		"<?php enum // c\nA {}",
		"<?php enum # c\nA {}",
		"<?php enum Extended {}",
		// case names may be reserved words, as class constant names
		"<?php enum T { case List; case ARRAY; case elseif; case Private; case or; }",
	}

	for _, v := range []string{"8.1", "8.5"} {
		ver, _ := version.New(v)
		for _, src := range append(names, decls...) {
			if r := parseWithErrs(parser.Parse, []byte(src), ver, false); len(r.errs) > 0 {
				t.Errorf("%s %q: %v", v, src, r.errs)
			}
		}
		for _, src := range decls {
			r := parseWithErrs(parser.Parse, []byte(src), ver, false)
			if root, ok := r.root.(*ast.Root); !ok || len(root.Stmts) != 1 {
				t.Errorf("%s %q: no statement", v, src)
			} else if _, ok := root.Stmts[0].(*ast.StmtEnum); !ok {
				t.Errorf("%s %q: got %T, want *ast.StmtEnum", v, src, root.Stmts[0])
			}
		}
	}
}
