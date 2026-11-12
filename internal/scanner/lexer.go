package scanner

import (
	"bytes"

	"github.com/pyxsoft/php-parser/pkg/conf"
	"github.com/pyxsoft/php-parser/pkg/errors"
	"github.com/pyxsoft/php-parser/pkg/position"
	"github.com/pyxsoft/php-parser/pkg/token"
	"github.com/pyxsoft/php-parser/pkg/version"
)

type Lexer struct {
	data           []byte
	phpVersion     *version.Version
	errHandlerFunc func(*errors.Error)

	p, pe, cs   int
	ts, te, act int
	stack       []int
	top         int

	heredocLabel     []byte
	heredocSince73   bool
	skipFreeFloating bool
	tokenPool        *token.Pool
	positionPool     *position.Pool
	newLines         NewLines
}

// Average source bytes per token and per line, used to size the initial
// pools so that small inputs do not pay for full blocks.
const (
	tokenSizeHint = 4
	lineSizeHint  = 40
)

func NewLexer(data []byte, config conf.Config) *Lexer {
	lex := &Lexer{
		data:           data,
		phpVersion:     config.Version,
		errHandlerFunc: config.ErrorHandlerFunc,

		pe:    len(data),
		stack: make([]int, 0),

		heredocSince73:   config.Version == nil || config.Version.GreaterOrEqual(&version.Version{Major: 7, Minor: 3}),
		skipFreeFloating: config.SkipFreeFloating,

		tokenPool:    token.NewPoolFor(len(data)/tokenSizeHint, token.DefaultBlockSize),
		positionPool: position.NewPoolFor(len(data)/tokenSizeHint, position.DefaultBlockSize),
		newLines:     NewLines{make([]int, 0, max(16, len(data)/lineSizeHint))},
	}

	initLexer(lex)
	if config.StartInPHP {
		lex.cs = lexer_en_php
	}

	return lex
}

func (lex *Lexer) setTokenPosition(token *token.Token) {
	pos := lex.positionPool.Get()

	pos.StartLine = lex.newLines.GetLine(lex.ts)
	pos.EndLine = lex.newLines.GetLine(lex.te - 1)
	pos.StartPos = lex.ts
	pos.EndPos = lex.te

	token.Position = pos
}

func (lex *Lexer) addFreeFloatingToken(t *token.Token, id token.ID, ps, pe int) {
	if lex.skipFreeFloating {
		return
	}

	skippedTkn := lex.tokenPool.Get()
	skippedTkn.ID = id
	skippedTkn.Value = lex.data[ps:pe]

	lex.setTokenPosition(skippedTkn)

	if t.FreeFloating == nil {
		t.FreeFloating = make([]*token.Token, 0, 2)
	}

	t.FreeFloating = append(t.FreeFloating, skippedTkn)
}

// at returns data[i], or 0 when i is out of range. Lookahead helpers run on
// untrusted input and must never index past either end of it.
func (lex *Lexer) at(i int) byte {
	if i < 0 || i >= len(lex.data) {
		return 0
	}

	return lex.data[i]
}

func (lex *Lexer) isNotStringVar() bool {
	p := lex.p
	if lex.at(p-1) == '\\' && lex.at(p-2) != '\\' {
		return true
	}

	if lex.at(p) == '$' && (lex.at(p+1) == '{' || isValidVarNameStart(lex.at(p+1))) {
		return false
	}

	if lex.at(p) == '{' && lex.at(p+1) == '$' {
		return false
	}

	return true
}

func (lex *Lexer) isNotStringEnd(s byte) bool {
	p := lex.p
	if lex.at(p-1) == '\\' && lex.at(p-2) != '\\' {
		return true
	}

	return lex.at(p) != s
}

func (lex *Lexer) isHeredocEnd(p int) bool {
	if lex.heredocSince73 {
		return lex.isHeredocEndSince73(p)
	}

	return lex.isHeredocEndBefore73(p)
}

func (lex *Lexer) isHeredocEndBefore73(p int) bool {
	if lex.at(p-1) != '\r' && lex.at(p-1) != '\n' {
		return false
	}

	l := len(lex.heredocLabel)
	if len(lex.data) < p+l {
		return false
	}

	if len(lex.data) > p+l && lex.data[p+l] != ';' && lex.data[p+l] != '\r' && lex.data[p+l] != '\n' {
		return false
	}

	if len(lex.data) > p+l+1 && lex.data[p+l] == ';' && lex.data[p+l+1] != '\r' && lex.data[p+l+1] != '\n' {
		return false
	}

	return bytes.Equal(lex.heredocLabel, lex.data[p:p+l])
}

func (lex *Lexer) isHeredocEndSince73(p int) bool {
	if lex.at(p-1) != '\r' && lex.at(p-1) != '\n' {
		return false
	}

	if p >= len(lex.data) {
		return false
	}

	for lex.at(p) == ' ' || lex.at(p) == '\t' {
		p++
	}

	l := len(lex.heredocLabel)
	if len(lex.data) < p+l {
		return false
	}

	if len(lex.data) > p+l && isValidVarName(lex.data[p+l]) {
		return false
	}

	if bytes.Equal(lex.heredocLabel, lex.data[p:p+l]) {
		lex.p = p
		return true
	}

	return false
}

// heredocEndAhead reports whether the closing label starts at p without
// moving the cursor (isHeredocEnd skips indentation by advancing lex.p).
func (lex *Lexer) heredocEndAhead(p int) bool {
	saved := lex.p
	ok := lex.isHeredocEnd(p)
	lex.p = saved

	return ok
}

func (lex *Lexer) isNotHeredocEnd(p int) bool {
	return !lex.isHeredocEnd(p)
}

func (lex *Lexer) growCallStack() {
	if lex.top == len(lex.stack) {
		lex.stack = append(lex.stack, 0)
	}
}

func (lex *Lexer) isNotPhpCloseToken() bool {
	return lex.at(lex.p) != '?' || lex.at(lex.p+1) != '>'
}

func (lex *Lexer) isNotNewLine() bool {
	if lex.at(lex.p) == '\n' && lex.at(lex.p-1) == '\r' {
		return true
	}

	return lex.at(lex.p-1) != '\n' && lex.at(lex.p-1) != '\r'
}

func (lex *Lexer) call(state int, fnext int) {
	lex.growCallStack()

	lex.stack[lex.top] = state
	lex.top++

	lex.p++
	lex.cs = fnext
}

func (lex *Lexer) ret(n int) {
	lex.p++

	// An unbalanced '}' has no state to return to: stay in the current state,
	// as php-src does, instead of indexing past the call stack.
	if lex.top < n {
		lex.top = 0
		return
	}

	lex.top = lex.top - n
	lex.cs = lex.stack[lex.top]
}

func (lex *Lexer) ungetStr(s string) {
	if bytes.HasSuffix(lex.data[lex.ts:lex.te], []byte(s)) {
		lex.ungetCnt(len(s))
	}
}

func (lex *Lexer) ungetCnt(n int) {
	lex.p = lex.p - n
	lex.te = lex.te - n
}

func (lex *Lexer) error(msg string) {
	if lex.errHandlerFunc == nil {
		return
	}

	pos := position.NewPosition(
		lex.newLines.GetLine(lex.ts),
		lex.newLines.GetLine(lex.te-1),
		lex.ts,
		lex.te,
	)

	lex.errHandlerFunc(errors.NewError(msg, pos))
}

func isValidVarNameStart(r byte) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_' || r >= 0x80
}

func isValidVarName(r byte) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r >= 0x80
}
