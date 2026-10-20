package conf

import (
	"github.com/pyxsoft/php-parser/pkg/errors"
	"github.com/pyxsoft/php-parser/pkg/version"
)

type Config struct {
	Version          *version.Version
	ErrorHandlerFunc func(e *errors.Error)
}
