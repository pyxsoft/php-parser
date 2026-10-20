package conf

import (
	"github.com/pyxsoft/php-parser/pkg/errors"
	"github.com/pyxsoft/php-parser/pkg/version"
)

type Config struct {
	Version          *version.Version
	ErrorHandlerFunc func(e *errors.Error)

	// SkipFreeFloating drops whitespace, comments and open tags instead of
	// attaching them to tokens as FreeFloating. It saves roughly half of the
	// lexer allocations for consumers that only inspect the AST; the printer
	// can no longer reproduce the original source exactly.
	SkipFreeFloating bool
}
