package parser_test

import (
	"bytes"
	"testing"

	"github.com/pyxsoft/php-parser/pkg/conf"
	"github.com/pyxsoft/php-parser/pkg/errors"
	"github.com/pyxsoft/php-parser/pkg/parser"
	"github.com/pyxsoft/php-parser/pkg/version"
	"github.com/pyxsoft/php-parser/pkg/visitor/dumper"
)

// TestStartInPHP checks that code lexed as PHP from its first byte gives the tree of the same
// code after "<?php ", in every grammar.
func TestStartInPHP(t *testing.T) {
	codes := []string{
		`echo 1; $a = "x{$b['c']}" . 'y';`,
		"function f($a) { return $a; } ?>html<?php echo f(2);",
		"$x = <<<EOT\n  {$a} b\n  EOT;\nif ($x) { eval($x); }",
		"echo 1 ?>",
		"syntax error ((",
		"",
	}
	dump := func(src []byte, v *version.Version, start bool) (string, int) {
		errs := 0
		root, err := parser.Parse(src, conf.Config{Version: v, SkipFreeFloating: true, StartInPHP: start,
			ErrorHandlerFunc: func(*errors.Error) { errs++ }})
		if err != nil {
			t.Fatal(err)
		}
		var b bytes.Buffer
		root.Accept(dumper.NewDumper(&b).WithTokens())
		return b.String(), errs
	}
	for _, v := range []*version.Version{{Major: 5, Minor: 6}, {Major: 7, Minor: 4}, php85} {
		for _, code := range codes {
			want, wantErrs := dump([]byte("<?php "+code), v, false)
			got, gotErrs := dump([]byte(code), v, true)
			if got != want || gotErrs != wantErrs {
				t.Errorf("%d.%d %q: StartInPHP gives another tree (errors %d, want %d)\n%s\nwant:\n%s", v.Major, v.Minor, code, gotErrs, wantErrs, got, want)
			}
		}
	}
}
