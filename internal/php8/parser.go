package php8

import (
	"github.com/pyxsoft/php-parser/internal/position"
	"github.com/pyxsoft/php-parser/pkg/ast"
	"github.com/pyxsoft/php-parser/pkg/conf"
	"github.com/pyxsoft/php-parser/pkg/errors"
	"github.com/pyxsoft/php-parser/pkg/token"
)

// nodeSizeHint is the average number of source bytes per AST node.
const nodeSizeHint = 8

// Parser structure
type Parser struct {
	Lexer          *Lexer
	currentToken   *token.Token
	rootNode       ast.Vertex
	errHandlerFunc func(*errors.Error)
	builder        *Builder
}

// NewParser creates and returns new Parser
func NewParser(lexer *Lexer, config conf.Config) *Parser {
	p := &Parser{
		Lexer:          lexer,
		errHandlerFunc: config.ErrorHandlerFunc,
	}
	p.builder = NewBuilder(position.NewBuilderFor(len(lexer.data)/nodeSizeHint), p)
	return p
}

// NewArenaParser returns a Parser whose AST nodes and positions come from a;
// lexer should come from NewArenaLexer with the same Arena. The tree stays
// valid until a.Reset.
func NewArenaParser(lexer *Lexer, config conf.Config, a *Arena) *Parser {
	p := &Parser{
		Lexer:          lexer,
		errHandlerFunc: config.ErrorHandlerFunc,
	}
	p.builder = NewBuilder(position.NewBuilderWithPool(a.nodePos), p)
	p.builder.alloc = &a.nodes
	return p
}

func (p *Parser) Lex(lval *yySymType) int {
	t := p.Lexer.Lex()

	p.currentToken = t
	lval.token = t

	return int(t.ID)
}

func (p *Parser) Error(msg string) {
	if p.errHandlerFunc == nil {
		return
	}

	p.errHandlerFunc(errors.NewError(msg, p.currentToken.Position))
}

// Parse the php7 Parser entrypoint
func (p *Parser) Parse() int {
	p.rootNode = nil

	return yyParse(p)
}

// GetRootNode returns root node
func (p *Parser) GetRootNode() ast.Vertex {
	return p.rootNode
}

// helpers

func lastNode(nn []ast.Vertex) ast.Vertex {
	if len(nn) == 0 {
		return nil
	}
	return nn[len(nn)-1]
}
