package php8_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pyxsoft/php-parser/internal/php8"
	"github.com/pyxsoft/php-parser/pkg/conf"
	"github.com/pyxsoft/php-parser/pkg/version"
)

// TestInterpolationTokens: a variable in a string takes one offset or one
// property, and what follows is text. Expected tokens come from token_get_all
// of PHP 8.4.
func TestInterpolationTokens(t *testing.T) {
	cases := []struct{ code, want string }{
		{`"$a->b[0]"`, `" T_VARIABLE($a) T_OBJECT_OPERATOR(->) T_STRING(b) T_ENCAPSED_AND_WHITESPACE([0]) "`},
		{`"$a->b[$k]x"`, `" T_VARIABLE($a) T_OBJECT_OPERATOR(->) T_STRING(b) T_ENCAPSED_AND_WHITESPACE([) T_VARIABLE($k) T_ENCAPSED_AND_WHITESPACE(]x) "`},
		{`"$a->b->c"`, `" T_VARIABLE($a) T_OBJECT_OPERATOR(->) T_STRING(b) T_ENCAPSED_AND_WHITESPACE(->c) "`},
		{`"$a?->b[1]"`, `" T_VARIABLE($a) T_NULLSAFE_OBJECT_OPERATOR(?->) T_STRING(b) T_ENCAPSED_AND_WHITESPACE([1]) "`},
		{`"$a[0]->b"`, `" T_VARIABLE($a) [ T_NUM_STRING(0) ] T_ENCAPSED_AND_WHITESPACE(->b) "`},
		{`"$a[k][j]"`, `" T_VARIABLE($a) [ T_STRING(k) ] T_ENCAPSED_AND_WHITESPACE([j]) "`},
		{`"$a->b$c->d[e]"`, `" T_VARIABLE($a) T_OBJECT_OPERATOR(->) T_STRING(b) T_VARIABLE($c) T_OBJECT_OPERATOR(->) T_STRING(d) T_ENCAPSED_AND_WHITESPACE([e]) "`},
		{"<<<EOT\n$a->b[0]\nEOT", "T_START_HEREDOC(<<<EOT\n) T_VARIABLE($a) T_OBJECT_OPERATOR(->) T_STRING(b) T_ENCAPSED_AND_WHITESPACE([0]\n) T_END_HEREDOC(EOT)"},
		{"`$a->b[x]`", "` T_VARIABLE($a) T_OBJECT_OPERATOR(->) T_STRING(b) T_ENCAPSED_AND_WHITESPACE([x]) `"},
		{`"$a->"`, `" T_VARIABLE($a) T_ENCAPSED_AND_WHITESPACE(->) "`},
		{`"$a->1"`, `" T_VARIABLE($a) T_ENCAPSED_AND_WHITESPACE(->1) "`},
	}

	v, _ := version.New("8.4")
	for _, c := range cases {
		lexer := php8.NewLexer([]byte("<?php "+c.code+";"), conf.Config{Version: v})
		var got []string
		for tkn := lexer.Lex(); tkn.ID != 0 && tkn.ID != ';'; tkn = lexer.Lex() {
			if tkn.ID < 256 {
				got = append(got, string(tkn.Value))
			} else {
				got = append(got, fmt.Sprintf("%s(%s)", tkn.ID, tkn.Value))
			}
		}
		if s := strings.Join(got, " "); s != c.want {
			t.Errorf("%s\n got: %s\nwant: %s", c.code, s, c.want)
		}
	}
}
