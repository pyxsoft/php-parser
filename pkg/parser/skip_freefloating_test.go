package parser_test

import (
	"reflect"
	"testing"

	"github.com/pyxsoft/php-parser/internal/benchcorpus"
	"github.com/pyxsoft/php-parser/pkg/ast"
	"github.com/pyxsoft/php-parser/pkg/conf"
	"github.com/pyxsoft/php-parser/pkg/parser"
	"github.com/pyxsoft/php-parser/pkg/token"
)

// walkTokens calls fn for every token reachable from v.
func walkTokens(v reflect.Value, fn func(*token.Token)) {
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer:
		if v.IsNil() {
			return
		}
		if t, ok := v.Interface().(*token.Token); ok {
			fn(t)
			return
		}
		walkTokens(v.Elem(), fn)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			walkTokens(v.Field(i), fn)
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			walkTokens(v.Index(i), fn)
		}
	}
}

func clearFreeFloating(root ast.Vertex) {
	walkTokens(reflect.ValueOf(root), func(t *token.Token) { t.FreeFloating = nil })
}

func hasFreeFloating(root ast.Vertex) bool {
	found := false
	walkTokens(reflect.ValueOf(root), func(t *token.Token) { found = found || t.FreeFloating != nil })
	return found
}

func TestSkipFreeFloatingKeepsAST(t *testing.T) {
	for _, f := range benchcorpus.Synthetic() {
		t.Run(f.Name, func(t *testing.T) {
			full, err := parser.Parse(f.Src, conf.Config{Version: benchVersion})
			if err != nil {
				t.Fatal(err)
			}
			if !hasFreeFloating(full) {
				t.Fatal("expected free-floating tokens in full mode")
			}
			clearFreeFloating(full)

			skip, err := parser.Parse(f.Src, conf.Config{Version: benchVersion, SkipFreeFloating: true})
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(full, skip) {
				t.Fatal("AST differs between full and SkipFreeFloating modes")
			}
			if hasFreeFloating(skip) {
				t.Fatal("SkipFreeFloating left free-floating tokens in the AST")
			}
		})
	}
}
